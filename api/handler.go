package api

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"vpn-manager/core/config"
	"vpn-manager/peers"
	"vpn-manager/servers"

	"github.com/gorilla/mux"
)

type IPeersService interface {
	GetByID(ctx context.Context, peerID string) (peers.Peer, error)
}

type IServersService interface {
	RegisterNewPeer(ctx context.Context, userID int64, serverID string) (servers.RegisterNewPeerOutput, error)
}

type IBot interface {
	SendLocationsList(userID int64) error
	SendPostImportInstructions(userID int64) error
}

type Handler struct {
	peersService   IPeersService
	serversService IServersService
	bot            IBot
	apiUrl         string
	apps           config.Apps
}

func NewHandler(peersService IPeersService, serversService IServersService, bot IBot, apiUrl string, apps config.Apps) *Handler {
	return &Handler{
		peersService:   peersService,
		serversService: serversService,
		bot:            bot,
		apiUrl:         apiUrl,
		apps:           apps,
	}
}

func (h *Handler) RegisterRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/peers/{id}/conf", h.getPeerConf).Methods("GET")
	r.HandleFunc("/setup", h.setup).Methods("GET")
	r.HandleFunc("/apps", h.downloadApp).Methods("GET")

	return r
}

func (h *Handler) downloadApp(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if err := h.bot.SendLocationsList(userID); err != nil {
		log.Print(err)
		http.Error(w, "Failed to send setup instructions", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, h.apps.AppStoreURL, http.StatusTemporaryRedirect)
}

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	peer, err := h.serversService.RegisterNewPeer(r.Context(), userID, query.Get("server_id"))
	if err != nil {
		http.Error(w, "failed to register new peer", http.StatusInternalServerError)
		return
	}

	if err := h.bot.SendPostImportInstructions(userID); err != nil {
		http.Error(w, "failed to send post import instructions", http.StatusInternalServerError)
		return
	}

	deep := fmt.Sprintf("streisand://import/%s/peers/%s/conf?name=%s", h.apiUrl, peer.PeerID, peer.Location)

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
