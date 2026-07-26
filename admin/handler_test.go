package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
	"vpn-manager/core/config"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/plans"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"github.com/gorilla/mux"
)

type stubUsers struct {
	blockedCalls []struct {
		userID  int64
		blocked bool
		reason  string
	}
}

func (s *stubUsers) GetByID(context.Context, int64) (users.User, error) {
	return users.User{ID: 1, Username: "tester"}, nil
}

func (s *stubUsers) List(context.Context, users.ListFilter) ([]users.User, int64, error) {
	return []users.User{{ID: 1, Username: "tester"}}, 1, nil
}

func (s *stubUsers) SetBlocked(_ context.Context, userID int64, blocked bool, reason string) error {
	s.blockedCalls = append(s.blockedCalls, struct {
		userID  int64
		blocked bool
		reason  string
	}{userID, blocked, reason})

	return nil
}

func (s *stubUsers) Totals(context.Context) (users.Totals, error) {
	return users.Totals{Total: 10, Blocked: 1}, nil
}

func (s *stubUsers) SignupsByDay(context.Context, time.Time) ([]users.DailyCount, error) {
	return []users.DailyCount{}, nil
}

type stubServers struct {
	healthCalled  bool
	deletedClient []string
}

func (s *stubServers) GetAll(context.Context) ([]servers.Server, error) {
	return []servers.Server{}, nil
}
func (s *stubServers) GetByID(context.Context, string) (servers.Server, error) {
	return servers.Server{}, servers.ErrServerNotFound
}

func (s *stubServers) Create(_ context.Context, input servers.CreateInput) (servers.Server, error) {
	if input.Location == "" {
		return servers.Server{}, servers.ErrInvalidInput
	}

	return servers.Server{ID: "srv1", Location: input.Location}, nil
}
func (s *stubServers) Update(context.Context, string, servers.UpdateInput) (servers.Server, error) {
	return servers.Server{}, servers.ErrServerNotFound
}
func (s *stubServers) Delete(context.Context, string) error { return servers.ErrServerNotFound }
func (s *stubServers) Count(context.Context) (int64, int64, error) {
	return 2, 1, nil
}
func (s *stubServers) CheckHealth(context.Context, string) (servers.Health, error) {
	return servers.Health{}, servers.ErrServerNotFound
}
func (s *stubServers) CheckAllHealth(context.Context) ([]servers.Health, error) {
	s.healthCalled = true
	return []servers.Health{}, nil
}
func (s *stubServers) DeletePeerFromServer(_ context.Context, _, email string) error {
	s.deletedClient = append(s.deletedClient, email)
	return nil
}
func (s *stubServers) RegisterNewPeers(context.Context, int64) error { return nil }

type stubPlans struct{}

func (stubPlans) GetAllIncludingInactive(context.Context) ([]plans.Plan, error) {
	return []plans.Plan{}, nil
}
func (stubPlans) GetByID(context.Context, string) (plans.Plan, error) {
	return plans.Plan{}, plans.ErrPlanNotFound
}
func (stubPlans) Create(context.Context, plans.CreateInput) (plans.Plan, error) {
	return plans.Plan{}, nil
}
func (stubPlans) Update(context.Context, string, plans.UpdateInput) (plans.Plan, error) {
	return plans.Plan{}, plans.ErrPlanNotFound
}
func (stubPlans) Delete(context.Context, string) error { return plans.ErrPlanNotFound }

type stubSubs struct{}

func (stubSubs) GetByUserID(context.Context, int64) (subscriptions.Subscription, error) {
	return subscriptions.Subscription{}, subscriptions.ErrSubscriptionNotFound
}
func (stubSubs) GetSubscriptionsForUsers(context.Context, []int64) (map[int64]subscriptions.Subscription, error) {
	return map[int64]subscriptions.Subscription{}, nil
}
func (stubSubs) Totals(context.Context) (subscriptions.Totals, error) {
	return subscriptions.Totals{Active: 5}, nil
}
func (stubSubs) CountByPlan(context.Context) ([]subscriptions.PlanCount, error) {
	return []subscriptions.PlanCount{}, nil
}
func (stubSubs) CreatedByDay(context.Context, time.Time, *bool) ([]subscriptions.DailyCount, error) {
	return []subscriptions.DailyCount{}, nil
}
func (stubSubs) Deactivate(context.Context, int64) error { return nil }

type stubPayments struct{}

func (stubPayments) List(context.Context, payments.ListFilter) ([]payments.Invoice, int64, error) {
	return []payments.Invoice{}, 0, nil
}
func (stubPayments) GetByUserID(context.Context, int64, int) ([]payments.Invoice, error) {
	return []payments.Invoice{}, nil
}
func (stubPayments) Totals(context.Context) (payments.Totals, error) {
	return payments.Totals{Revenue: 100}, nil
}
func (stubPayments) RevenueSince(context.Context, time.Time) (float64, error) { return 50, nil }
func (stubPayments) RevenueByDay(context.Context, time.Time) ([]payments.DailyRevenue, error) {
	return []payments.DailyRevenue{}, nil
}
func (stubPayments) RevenueByPlan(context.Context, time.Time) ([]payments.PlanRevenue, error) {
	return []payments.PlanRevenue{}, nil
}

type stubPeers struct{ deactivated []int64 }

func (s *stubPeers) GetPeerByUserID(context.Context, int64) (peers.Peer, error) {
	return peers.Peer{
		UserID: 1,
		UUID:   "uuid-1",
		Email:  "abc1234",
		Subs:   []peers.Sub{{Location: "AMS", ServerID: "srv1", Enabled: true}},
	}, nil
}
func (s *stubPeers) GetPeersForUsers(context.Context, []int64) (map[int64]peers.Peer, error) {
	return map[int64]peers.Peer{}, nil
}
func (s *stubPeers) ActivatePeer(context.Context, int64) error { return nil }
func (s *stubPeers) DeactivatePeer(_ context.Context, userID int64) error {
	s.deactivated = append(s.deactivated, userID)
	return nil
}
func (s *stubPeers) Totals(context.Context) (peers.Totals, error) {
	return peers.Totals{Total: 8, Imported: 4}, nil
}
func (s *stubPeers) CountByLocation(context.Context) ([]peers.LocationCount, error) {
	return []peers.LocationCount{}, nil
}

type nopLogger struct{}

func (nopLogger) Debug(...any)          {}
func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Info(...any)           {}
func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Warn(...any)           {}
func (nopLogger) Warnf(string, ...any)  {}
func (nopLogger) Error(...any)          {}
func (nopLogger) Errorf(string, ...any) {}

type fixture struct {
	router  *mux.Router
	users   *stubUsers
	servers *stubServers
	peers   *stubPeers
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	u := &stubUsers{}
	s := &stubServers{}
	p := &stubPeers{}

	handler := NewHandler(config.Admin{
		Username:       "root",
		Password:       "s3cret",
		JWTSecret:      testSecret,
		TokenTTL:       time.Hour,
		AllowedOrigins: []string{"http://localhost:5173"},
	}, Deps{
		Users:         u,
		Servers:       s,
		Plans:         stubPlans{},
		Subscriptions: stubSubs{},
		Payments:      stubPayments{},
		Peers:         p,
		Logger:        nopLogger{},
	})

	router := mux.NewRouter()
	handler.RegisterRoutes(router)

	return &fixture{router: router, users: u, servers: s, peers: p}
}

func (f *fixture) do(t *testing.T, method, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()

	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	req := httptest.NewRequest(method, path, reader)
	req.RemoteAddr = "10.0.0.1:1234"

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}

	rec := httptest.NewRecorder()
	f.router.ServeHTTP(rec, req)

	return rec
}

func (f *fixture) login(t *testing.T) string {
	t.Helper()

	rec := f.do(t, http.MethodPost, "/admin/api/v1/auth/login", "", `{"username":"root","password":"s3cret"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("login status = %d, body = %s", rec.Code, rec.Body)
	}

	var response loginResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode login response: %v", err)
	}

	if response.Token == "" {
		t.Fatal("login returned an empty token")
	}

	return response.Token
}

func TestLoginRejectsWrongPassword(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/auth/login", "", `{"username":"root","password":"nope"}`)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}

	if body := rec.Body.String(); !strings.Contains(body, "invalid username or password") {
		t.Errorf("body = %s, want a generic credentials error", body)
	}
}

func TestProtectedRoutesRequireToken(t *testing.T) {
	f := newFixture(t)

	protected := []struct{ method, path string }{
		{http.MethodGet, "/admin/api/v1/analytics/overview"},
		{http.MethodGet, "/admin/api/v1/analytics/timeseries"},
		{http.MethodGet, "/admin/api/v1/users"},
		{http.MethodGet, "/admin/api/v1/servers"},
		{http.MethodGet, "/admin/api/v1/plans"},
		{http.MethodGet, "/admin/api/v1/payments"},
		{http.MethodGet, "/admin/api/v1/audit"},
		{http.MethodPost, "/admin/api/v1/users/1/block"},
	}

	for _, route := range protected {
		rec := f.do(t, route.method, route.path, "", "")
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want 401", route.method, route.path, rec.Code)
		}
	}
}

func TestProtectedRoutesRejectTamperedToken(t *testing.T) {
	f := newFixture(t)
	token := f.login(t)

	tampered := token[:len(token)-1] + map[bool]string{true: "A", false: "B"}[strings.HasSuffix(token, "B")]

	rec := f.do(t, http.MethodGet, "/admin/api/v1/analytics/overview", tampered, "")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestOverviewReturnsAggregatedNumbers(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodGet, "/admin/api/v1/analytics/overview", f.login(t), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	var response overviewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if response.Users.Total != 10 {
		t.Errorf("users.total = %d, want 10", response.Users.Total)
	}

	if response.Servers.Active != 1 {
		t.Errorf("servers.active = %d, want 1", response.Servers.Active)
	}

	if response.Revenue.ARPU != 10 {
		t.Errorf("revenue.arpu = %v, want 10", response.Revenue.ARPU)
	}
}

func TestBlockUserRevokesAccess(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/users/1/block", f.login(t), `{"reason":"abuse"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	if len(f.users.blockedCalls) != 1 {
		t.Fatalf("SetBlocked calls = %d, want 1", len(f.users.blockedCalls))
	}

	call := f.users.blockedCalls[0]
	if call.userID != 1 || !call.blocked || call.reason != "abuse" {
		t.Errorf("SetBlocked(%d, %v, %q), want (1, true, \"abuse\")", call.userID, call.blocked, call.reason)
	}

	if len(f.peers.deactivated) != 1 || f.peers.deactivated[0] != 1 {
		t.Errorf("deactivated peers = %v, want [1]", f.peers.deactivated)
	}

	if len(f.servers.deletedClient) != 1 || f.servers.deletedClient[0] != "abc1234" {
		t.Errorf("deleted clients = %v, want [abc1234]", f.servers.deletedClient)
	}
}

func TestUnblockUserClearsFlag(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/users/1/unblock", f.login(t), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	if len(f.users.blockedCalls) != 1 || f.users.blockedCalls[0].blocked {
		t.Errorf("SetBlocked calls = %+v, want one call with blocked=false", f.users.blockedCalls)
	}
}

func TestInvalidUserIDIsRejected(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/users/not-a-number/block", f.login(t), "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestServersHealthRouteWinsOverIDRoute(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodGet, "/admin/api/v1/servers/health", f.login(t), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}

	if !f.servers.healthCalled {
		t.Error("CheckAllHealth was not called — the route resolved elsewhere")
	}
}

func TestCreateServerMapsValidationErrorTo400(t *testing.T) {
	f := newFixture(t)
	token := f.login(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/servers", token,
		`{"location":"","ip":"1.2.3.4","api_url":"http://x","auth_token":"t","port":443}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400, body = %s", rec.Code, rec.Body)
	}

	rec = f.do(t, http.MethodPost, "/admin/api/v1/servers", token,
		`{"location":"AMS","ip":"1.2.3.4","api_url":"http://x","auth_token":"t","port":443,"inbound_id":1,"is_active":true}`)
	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201, body = %s", rec.Code, rec.Body)
	}
}

func TestUnknownFieldsAreRejected(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodPost, "/admin/api/v1/servers", f.login(t),
		`{"location":"AMS","ip":"1.2.3.4","api_url":"http://x","auth_token":"t","port":443,"typo_field":1}`)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestCORSOnlyAnswersAllowedOrigins(t *testing.T) {
	f := newFixture(t)

	cases := []struct {
		origin string
		want   string
	}{
		{"http://localhost:5173", "http://localhost:5173"},
		{"http://evil.example", ""},
	}

	for _, c := range cases {
		req := httptest.NewRequest(http.MethodOptions, "/admin/api/v1/users", nil)
		req.Header.Set("Origin", c.origin)
		req.RemoteAddr = "10.0.0.1:1234"

		rec := httptest.NewRecorder()
		f.router.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != c.want {
			t.Errorf("origin %s: Allow-Origin = %q, want %q", c.origin, got, c.want)
		}
	}
}

func TestSecurityHeadersArePresent(t *testing.T) {
	f := newFixture(t)

	rec := f.do(t, http.MethodGet, "/admin/api/v1/analytics/overview", f.login(t), "")

	want := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"Cache-Control":          "no-store",
	}

	for header, value := range want {
		if got := rec.Header().Get(header); got != value {
			t.Errorf("%s = %q, want %q", header, got, value)
		}
	}
}
