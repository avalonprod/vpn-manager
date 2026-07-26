package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"vpn-manager/peers"
)

type stubPeers struct {
	byToken map[string]peers.Peer
}

func (s stubPeers) GetByAccessToken(_ context.Context, token string) (peers.Peer, error) {
	peer, ok := s.byToken[token]
	if !ok {
		return peers.Peer{}, peers.ErrPeerNotFound
	}
	return peer, nil
}

func (s stubPeers) GetActivePeerByUserID(context.Context, int64) (peers.Peer, error) {
	return peers.Peer{}, peers.ErrPeerNotFound
}

func (s stubPeers) SetImported(context.Context, int64) error { return nil }

func identifyWith(t *testing.T, legacy bool, target string) (int, int64, bool) {
	t.Helper()

	h := &Handler{
		logger:           nopLogger{},
		allowLegacyLinks: legacy,
		peersService: stubPeers{byToken: map[string]peers.Peer{
			"good-token": {UserID: 777},
		}},
	}

	var seen int64
	var identified bool

	guarded := h.Identify(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, identified = userIDFrom(r.Context())
	}))

	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	return rec.Code, seen, identified
}

func TestIdentifyResolvesUserFromToken(t *testing.T) {
	code, userID, ok := identifyWith(t, false, "/subs?token=good-token")

	if code != http.StatusOK || !ok {
		t.Fatalf("status = %d, identified = %v; want the token to resolve", code, ok)
	}

	if userID != 777 {
		t.Errorf("userID = %d, want 777", userID)
	}
}

func TestIdentifyRejectsUserIDWhenLegacyDisabled(t *testing.T) {
	code, _, ok := identifyWith(t, false, "/subs?user_id=12345")

	if ok {
		t.Fatal("a raw user_id identified the caller — the IDOR is still open")
	}

	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestIdentifyRejectsUnknownToken(t *testing.T) {
	code, _, ok := identifyWith(t, false, "/subs?token=someone-elses-guess")

	if ok {
		t.Fatal("an unknown token identified the caller")
	}

	if code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", code)
	}
}

func TestIdentifyIgnoresUserIDEvenAlongsideToken(t *testing.T) {
	code, userID, ok := identifyWith(t, false, "/subs?token=good-token&user_id=999999")

	if code != http.StatusOK || !ok {
		t.Fatalf("status = %d, identified = %v", code, ok)
	}

	if userID != 777 {
		t.Errorf("userID = %d, want 777 — user_id must never override the token", userID)
	}
}

func TestIdentifyAcceptsUserIDOnlyWhenLegacyEnabled(t *testing.T) {
	code, userID, ok := identifyWith(t, true, "/subs?user_id=12345")

	if code != http.StatusOK || !ok {
		t.Fatalf("status = %d, identified = %v; legacy mode must still work", code, ok)
	}

	if userID != 12345 {
		t.Errorf("userID = %d, want 12345", userID)
	}
}

type stubUsers struct{ blocked bool }

func (s stubUsers) IsBlocked(context.Context, int64) (bool, error) { return s.blocked, nil }

type stubSubs struct{ active bool }

func (s stubSubs) IsSubscriptionActive(context.Context, int64) (bool, error) {
	return s.active, nil
}

func (s stubSubs) CreateOrExtend(context.Context, int64, string) error { return nil }

func TestAccessGuardWorksWithTokenOnlyLinks(t *testing.T) {
	code, userID := guardedRequestVia(t, "access", "/setup?token=good-token&os=ios", false, true)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — a token-only link must pass AccessGuard", code)
	}

	if userID != 777 {
		t.Errorf("userID = %d, want 777", userID)
	}
}

func TestAccessGuardBlocksBlockedUser(t *testing.T) {
	code, _ := guardedRequestVia(t, "access", "/subs?token=good-token", true, true)

	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
}

func TestAccessGuardRejectsInactiveSubscription(t *testing.T) {
	code, _ := guardedRequestVia(t, "access", "/subs?token=good-token", false, false)

	if code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", code)
	}
}

func TestBlockGuardWorksWithTokenOnlyLinks(t *testing.T) {
	code, userID := guardedRequestVia(t, "block", "/apps?token=good-token&os=ios", false, false)

	if code != http.StatusOK {
		t.Fatalf("status = %d, want 200 — /apps must work without a subscription", code)
	}

	if userID != 777 {
		t.Errorf("userID = %d, want 777", userID)
	}
}

func guardedRequestVia(t *testing.T, kind, target string, blocked, active bool) (int, int64) {
	t.Helper()

	h := &Handler{
		logger:               nopLogger{},
		usersService:         stubUsers{blocked: blocked},
		subscriptionsService: stubSubs{active: active},
		peersService: stubPeers{byToken: map[string]peers.Peer{
			"good-token": {UserID: 777},
		}},
	}

	guard := h.BlockGuard
	if kind == "access" {
		guard = h.AccessGuard
	}

	var seen int64

	wrapped := guard(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, _ = userIDFrom(r.Context())
	}))

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))

	return rec.Code, seen
}

func TestIdentifyRejectsGarbageLegacyUserID(t *testing.T) {
	code, _, ok := identifyWith(t, true, "/subs?user_id=not-a-number")

	if ok {
		t.Fatal("a malformed user_id identified the caller")
	}

	if code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", code)
	}
}
