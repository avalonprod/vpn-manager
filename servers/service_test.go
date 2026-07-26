package servers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"vpn-manager/peers"

	"go.mongodb.org/mongo-driver/bson"
)

type updateSpy struct {
	IStore
	fields bson.M
}

func (s *updateSpy) Update(_ context.Context, _ string, fields bson.M) error {
	s.fields = fields
	return nil
}

func (s *updateSpy) GetByID(context.Context, string) (Server, error) {
	return Server{}, nil
}

func updateWith(t *testing.T, input UpdateInput) bson.M {
	t.Helper()

	spy := &updateSpy{}

	if _, err := (&service{store: spy}).Update(context.Background(), "srv1", input); err != nil {
		t.Fatalf("update: %v", err)
	}

	return spy.fields
}

func ptr[T any](v T) *T { return &v }

func TestUpdateKeepsExistingTokenWhenBlank(t *testing.T) {
	for name, token := range map[string]string{"empty": "", "blank": "   "} {
		fields := updateWith(t, UpdateInput{AuthToken: ptr(token)})

		if _, present := fields["auth_token"]; present {
			t.Errorf("%s token: auth_token = %v, want the field to be left untouched", name, fields["auth_token"])
		}
	}
}

func TestUpdateStoresTrimmedToken(t *testing.T) {
	fields := updateWith(t, UpdateInput{AuthToken: ptr("  new-token  ")})

	if got := fields["auth_token"]; got != "new-token" {
		t.Errorf("auth_token = %v, want %q", got, "new-token")
	}
}

func TestUpdateIgnoresOmittedFields(t *testing.T) {
	fields := updateWith(t, UpdateInput{Location: ptr("Berlin")})

	if len(fields) != 1 || fields["location"] != "Berlin" {
		t.Errorf("fields = %v, want only {location: Berlin}", fields)
	}
}

func callTestPanel(t *testing.T, handler http.HandlerFunc) error {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	server := Server{ID: "srv1", ApiUrl: srv.URL, AuthToken: "token"}

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/panel/api/clients/add", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	authorize(req, server)

	return callPanel(srv.Client(), req, server)
}

func TestCallPanelFailsOnUnsuccessfulEnvelope(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success":false,"msg":"duplicate email"}`))
	})

	if err == nil {
		t.Fatal("success=false in a 200 response was treated as success")
	}

	if !strings.Contains(err.Error(), "duplicate email") {
		t.Errorf("error %q does not carry the panel message", err)
	}
}

func TestCallPanelAcceptsSuccessfulEnvelope(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"msg":"","obj":null}`))
	})

	if err != nil {
		t.Errorf("successful response rejected: %v", err)
	}
}

func TestCallPanelReportsRejectedToken(t *testing.T) {
	err := callTestPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if err == nil || !strings.Contains(err.Error(), "rejected the auth token") {
		t.Errorf("err = %v, want a rejected-token error", err)
	}
}

type stubUsers struct {
	blocked bool
	err     error
}

func (s stubUsers) IsBlocked(context.Context, int64) (bool, error) {
	return s.blocked, s.err
}

type stubPeers struct {
	created  bool
	subsSet  [][]peers.Sub
	createFn func() (peers.Peer, error)
}

func (s *stubPeers) Create(context.Context, int64) (peers.Peer, error) {
	s.created = true
	if s.createFn != nil {
		return s.createFn()
	}
	return peers.Peer{ID: "507f1f77bcf86cd799439011", UUID: "uuid-1", Email: "abc1234"}, nil
}

func (s *stubPeers) UpdateSubs(_ context.Context, _ string, subs []peers.Sub) error {
	s.subsSet = append(s.subsSet, subs)
	return nil
}

type stubActiveStore struct {
	IStore
	servers []Server
}

func (s *stubActiveStore) GetAllActiveServers(context.Context) ([]Server, error) {
	return s.servers, nil
}

func TestRegisterNewPeersRefusesBlockedUser(t *testing.T) {
	p := &stubPeers{}

	s := &service{
		store:        &stubActiveStore{servers: []Server{{ID: "srv1", AuthToken: "t", ApiUrl: "http://unused"}}},
		peersService: p,
		usersService: stubUsers{blocked: true},
	}

	err := s.RegisterNewPeers(context.Background(), 42)

	if !errors.Is(err, ErrUserBlocked) {
		t.Fatalf("err = %v, want ErrUserBlocked", err)
	}

	if p.created {
		t.Error("peer was created for a blocked user")
	}

	if len(p.subsSet) != 0 {
		t.Errorf("subs were written for a blocked user: %v", p.subsSet)
	}
}

func TestRegisterNewPeersFailsWhenBlockCheckFails(t *testing.T) {
	s := &service{
		peersService: &stubPeers{},
		usersService: stubUsers{err: errors.New("mongo down")},
	}

	if err := s.RegisterNewPeers(context.Background(), 42); err == nil {
		t.Fatal("a failing block check was treated as not blocked")
	}
}

func registerAgainstPanel(t *testing.T, handler http.HandlerFunc) (*stubPeers, error) {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	p := &stubPeers{}

	s := &service{
		store: &stubActiveStore{servers: []Server{{
			ID:        "srv1",
			Location:  "Almaty",
			ApiUrl:    srv.URL,
			AuthToken: "token",
			InBoundID: 1,
		}}},
		peersService: p,
		usersService: stubUsers{},
	}

	return p, s.RegisterNewPeers(context.Background(), 42)
}

func TestRegisterNewPeersTreatsExistingClientAsSuccess(t *testing.T) {
	for _, msg := range []string{
		"Something went wrong (email already in use: 0a8e470)",
		"client already exists",
		"duplicate email",
	} {
		p, err := registerAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`{"success":false,"msg":"` + msg + `"}`))
		})
		if err != nil {
			t.Fatalf("%q: err = %v, want nil — the client is already on the panel", msg, err)
		}

		if len(p.subsSet) != 1 || len(p.subsSet[0]) != 1 {
			t.Fatalf("%q: subs = %v, want the server kept in the subscription", msg, p.subsSet)
		}

		if p.subsSet[0][0].Location != "Almaty" {
			t.Errorf("%q: location = %q, want Almaty", msg, p.subsSet[0][0].Location)
		}
	}
}

func TestRegisterNewPeersStillFailsOnRealRejections(t *testing.T) {
	p, err := registerAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"msg":"Inbound not found"}`))
	})

	if !errors.Is(err, ErrNoServersAvailable) {
		t.Errorf("err = %v, want ErrNoServersAvailable", err)
	}

	if len(p.subsSet) != 1 || len(p.subsSet[0]) != 0 {
		t.Errorf("subs = %v, want no subscription for a server that rejected us", p.subsSet)
	}
}

func TestRegisterNewPeersSucceedsOnCleanAdd(t *testing.T) {
	p, err := registerAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"msg":""}`))
	})
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	if len(p.subsSet) != 1 || len(p.subsSet[0]) != 1 {
		t.Errorf("subs = %v, want one entry", p.subsSet)
	}
}

func TestRegisterNewPeersReportsWhenNoServerAccepted(t *testing.T) {
	p := &stubPeers{}

	s := &service{
		store:        &stubActiveStore{servers: []Server{{ID: "srv1", AuthToken: ""}}},
		peersService: p,
		usersService: stubUsers{},
	}

	err := s.RegisterNewPeers(context.Background(), 42)

	if !errors.Is(err, ErrNoServersAvailable) {
		t.Errorf("err = %v, want ErrNoServersAvailable", err)
	}

	if len(p.subsSet) != 1 || len(p.subsSet[0]) != 0 {
		t.Errorf("subs = %v, want a single empty write so no stale config survives", p.subsSet)
	}
}

type stubServerStore struct {
	IStore
	server Server
}

func (s *stubServerStore) GetByID(context.Context, string) (Server, error) {
	return s.server, nil
}

func deleteAgainstPanel(t *testing.T, handler http.HandlerFunc) error {
	t.Helper()

	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	s := &service{store: &stubServerStore{server: Server{
		ID:        "srv1",
		ApiUrl:    srv.URL,
		AuthToken: "token",
		InBoundID: 1,
	}}}

	return s.DeletePeerFromServer(context.Background(), "srv1", "abc1234")
}

func TestDeletePeerFailsWhenEndpointMissing(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	if err == nil {
		t.Fatal("a 404 from the panel was reported as a successful revocation")
	}

	if !errors.Is(err, ErrPanelEndpointMissing) {
		t.Errorf("err = %v, want ErrPanelEndpointMissing", err)
	}
}

func TestDeletePeerFailsWhenTokenRejected(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})

	if !errors.Is(err, ErrPanelUnauthorized) {
		t.Errorf("err = %v, want ErrPanelUnauthorized", err)
	}
}

func TestDeletePeerFailsOnInboundMisconfiguration(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"msg":"Inbound not found"}`))
	})

	if err == nil {
		t.Fatal("a panel misconfiguration was counted as a successful revocation")
	}
}

func TestDeletePeerSucceedsWhenClientAlreadyGone(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"msg":"client not found"}`))
	})

	if err != nil {
		t.Errorf("err = %v, want nil for an already-removed client", err)
	}
}

func TestDeletePeerFailsOnOtherPanelRejections(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":false,"msg":"database is locked"}`))
	})

	if err == nil {
		t.Fatal("an unrelated panel rejection was treated as success")
	}
}

func TestDeletePeerSucceedsOnOK(t *testing.T) {
	err := deleteAgainstPanel(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"success":true,"msg":""}`))
	})

	if err != nil {
		t.Errorf("err = %v, want nil", err)
	}
}

func TestAuthorizeSetsBearerAndAjaxHeaders(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://panel.example", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	authorize(req, Server{AuthToken: "secret-token"})

	if got := req.Header.Get("Authorization"); got != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", got, "Bearer secret-token")
	}

	if got := req.Header.Get("X-Requested-With"); got != "XMLHttpRequest" {
		t.Errorf("X-Requested-With = %q, want XMLHttpRequest", got)
	}
}

func validInput() CreateInput {
	return CreateInput{
		Location:  "Amsterdam",
		AuthToken: "3x-ui=session-value",
		Ip:        "203.0.113.10",
		ApiUrl:    "http://203.0.113.10:2053",
		Port:      443,
		InBoundID: 1,
	}
}

func TestValidateCreateInputAcceptsValidServer(t *testing.T) {
	if err := validateCreateInput(validInput()); err != nil {
		t.Fatalf("valid input rejected: %v", err)
	}
}

func TestValidateCreateInputRejectsBadValues(t *testing.T) {
	cases := map[string]func(*CreateInput){
		"empty location":   func(in *CreateInput) { in.Location = "  " },
		"empty ip":         func(in *CreateInput) { in.Ip = "" },
		"empty api url":    func(in *CreateInput) { in.ApiUrl = "" },
		"empty auth token": func(in *CreateInput) { in.AuthToken = "" },
		"blank auth token": func(in *CreateInput) { in.AuthToken = "   " },
		"zero port":        func(in *CreateInput) { in.Port = 0 },
		"port too large":   func(in *CreateInput) { in.Port = 70000 },
		"negative limit":   func(in *CreateInput) { in.MaxClients = -1 },
		"negative port":    func(in *CreateInput) { in.Port = -1 },
	}

	for name, mutate := range cases {
		input := validInput()
		mutate(&input)

		err := validateCreateInput(input)
		if err == nil {
			t.Errorf("%s: expected an error, got nil", name)
			continue
		}

		if !errors.Is(err, ErrInvalidInput) {
			t.Errorf("%s: error %v does not wrap ErrInvalidInput", name, err)
		}
	}
}
