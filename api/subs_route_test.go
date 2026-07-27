package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"vpn-manager/peers"

	"github.com/gorilla/mux"
)

func routerWithIdentify(t *testing.T) (*mux.Router, *int64) {
	t.Helper()

	h := &Handler{
		logger: nopLogger{},
		peersService: stubPeers{byToken: map[string]peers.Peer{
			"good-token": {UserID: 777},
		}},
	}

	var seen int64

	sink := h.Identify(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		seen, _ = userIDFrom(r.Context())
	}))

	router := mux.NewRouter()
	router.Handle("/subs/{token}", sink).Methods(http.MethodGet)
	router.Handle("/subs", sink).Methods(http.MethodGet)

	return router, &seen
}

func TestIdentifyResolvesTokenFromPath(t *testing.T) {
	router, seen := routerWithIdentify(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/subs/good-token", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if *seen != 777 {
		t.Errorf("userID = %d, want 777", *seen)
	}
}

// Уже импортированные приложения ходят по старой ссылке с query — она обязана
// продолжать работать, иначе у существующих пользователей отвалится обновление.
func TestIdentifyStillResolvesTokenFromQuery(t *testing.T) {
	router, seen := routerWithIdentify(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/subs?token=good-token", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	if *seen != 777 {
		t.Errorf("userID = %d, want 777", *seen)
	}
}

func TestUnknownTokenInPathIsRejected(t *testing.T) {
	router, _ := routerWithIdentify(t)

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/subs/nope", nil))

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}
