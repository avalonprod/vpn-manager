package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"vpn-manager/core/config"
	"vpn-manager/peers"

	"github.com/gorilla/mux"
)

type IPeersService interface {
	GetPeersByUserID(ctx context.Context, userID int64) ([]peers.Peer, error)
}

type IServersService interface {
	RegisterNewPeers(ctx context.Context, userID int64) ([]string, error)
}

type IBot interface {
	SendSetupInstruction(userID int64, os string) error
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

	r.HandleFunc("/subs", h.getSubs).Methods("GET")
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

	if err := h.bot.SendSetupInstruction(userID, query.Get("os")); err != nil {
		log.Print(err)
		http.Error(w, "Failed to send setup instructions", http.StatusInternalServerError)
		return
	}

	var appUrl string
	switch query.Get("os") {
	case "ios":
		appUrl = h.apps.AppStoreURL
	case "macos":
		appUrl = h.apps.AppStoreURL
	case "android":
		appUrl = h.apps.PlayMarketURL
	}

	http.Redirect(w, r, appUrl, http.StatusTemporaryRedirect)
}

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if err := h.bot.SendPostImportInstructions(userID); err != nil {
		http.Error(w, "failed to send post import instructions", http.StatusInternalServerError)
		return
	}

	var deep string

	switch query.Get("os") {
	case "ios":
		deep = fmt.Sprintf("streisand://import/%s/subs?user_id=%d&name=%s", h.apiUrl, userID, "NeonGuard")
	case "macos":
		deep = fmt.Sprintf("streisand://import/%s/subs?user_id=%d&name=%s", h.apiUrl, userID, "NeonGuard")
	case "android":
		deep = fmt.Sprintf("v2raytun://import/%s/subs?user_id=%d&name=%s", h.apiUrl, userID, "NeonGuard")
	}

	http.Redirect(w, r, deep, http.StatusTemporaryRedirect)
}

func (h *Handler) getSubs(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	peers, err := h.serversService.RegisterNewPeers(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to register new peer", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, strings.Join(peers, "\n"))
}
