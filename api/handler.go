package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"vpn-manager/peers"

	"github.com/gorilla/mux"
)

type IPeersService interface {
	GetByID(ctx context.Context, peerID string) (peers.Peer, error)
}

type Handler struct {
	peersService IPeersService
	apiUrl       string
}

func NewHandler(peersService IPeersService, apiUrl string) *Handler {
	return &Handler{
		peersService: peersService,
		apiUrl:       apiUrl,
	}
}

func (h *Handler) RegisterRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/peers/{id}", h.getPeer).Methods("GET")
	r.HandleFunc("/peers/{id}/conf", h.getPeerConf).Methods("GET")

	return r
}

func (h *Handler) getPeer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	peer, err := h.peersService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, peers.ErrPeerNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternalServerError, http.StatusInternalServerError)
		return
	}

	deep := fmt.Sprintf("streisand://import/%s/peers/%s/conf?name=%s", h.apiUrl, peer.ID, peer.Location)

	http.Redirect(w, r, deep, http.StatusTemporaryRedirect)
}

func (h *Handler) getPeerConf(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	peer, err := h.peersService.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, peers.ErrPeerNotFound) {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		http.Error(w, ErrInternalServerError, http.StatusInternalServerError)
		return
	}

	subs := map[string]interface{}{
		"version": 1,
		"peers": []map[string]string{
			{
				"name":   peer.Location,
				"type":   "wireguard",
				"config": peer.Config,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="config.subs"`)
	fmt.Fprint(w, subs)
}
