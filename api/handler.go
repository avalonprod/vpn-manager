package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"vpn-manager/core/config"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/plans"

	"github.com/gorilla/mux"
)

type IPeersService interface {
	GetActivePeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
}

type IPlansService interface {
	GetByID(ctx context.Context, ID string) (plans.Plan, error)
}

type IPaymentsService interface {
	CreateInvoice(ctx context.Context, input payments.CreateInvoiceInput) (string, error)
	GetInvoiceByID(ctx context.Context, userID int64, ID string) (payments.Invoice, error)
}

type ISubscriptionsService interface {
	CreateOrExtend(ctx context.Context, userID int64, invoiceID string) error
}

type IBot interface {
	SendSetupInstruction(userID int64, os string) error
	SendPostImportInstructions(userID int64, os string) error
	SendSuccessPayment(userID int64) error
}

type Handler struct {
	peersService         IPeersService
	plansService         IPlansService
	paymentsService      IPaymentsService
	subscriptionsService ISubscriptionsService
	bot                  IBot
	apiUrl               string
	apps                 config.Apps
}

func NewHandler(peersService IPeersService, plansService IPlansService, paymentsService IPaymentsService, subscriptionsService ISubscriptionsService, bot IBot, apiUrl string, apps config.Apps) *Handler {
	return &Handler{
		peersService:         peersService,
		plansService:         plansService,
		paymentsService:      paymentsService,
		subscriptionsService: subscriptionsService,
		bot:                  bot,
		apiUrl:               apiUrl,
		apps:                 apps,
	}
}

func (h *Handler) RegisterRoutes() *mux.Router {
	r := mux.NewRouter()

	r.HandleFunc("/subs", h.getSubs).Methods("GET")
	r.HandleFunc("/setup", h.setup).Methods("GET")
	r.HandleFunc("/apps", h.downloadApp).Methods("GET")
	r.HandleFunc("/subscribe", h.handleSubscribe).Methods("GET")
	r.HandleFunc("/cloudpayments/webhook/check", h.handleCheckCloudPaymentsWebhook).Methods("GET")
	r.HandleFunc("/cloudpayments/webhook/pay", h.handlePayCloudPaymentsWebhook).Methods("POST")

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

	if err := h.bot.SendPostImportInstructions(userID, query.Get("os")); err != nil {
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

	peer, err := h.peersService.GetActivePeerByUserID(r.Context(), userID)
	if err != nil {
		http.Error(w, "failed to get subs", http.StatusInternalServerError)
		return
	}

	subs := make([]string, 0, len(peer.Subs))

	for _, sub := range peer.Subs {
		subs = append(subs, sub.URL)
	}

	fmt.Fprint(w, strings.Join(subs, "\n"))
}

func (h *Handler) handleCheckCloudPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	accountId := query.Get("AccountId")
	invoiceId := query.Get("InvoiceId")

	userID, err := strconv.ParseInt(accountId, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	_, err = h.paymentsService.GetInvoiceByID(r.Context(), userID, invoiceId)
	if err != nil {
		w.Write([]byte(`{"code":13}`))
		return
	}

	w.Write([]byte(`{"code":0}`))
}

type PayRequest struct {
	TransactionId string  `json:"TransactionId"`
	Amount        float64 `json:"Amount"`
	InvoiceId     string  `json:"InvoiceId"`
	AccountId     string  `json:"AccountId"`
}

func (h *Handler) handlePayCloudPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	accountID := r.FormValue("AccountId")
	invoiceID := r.FormValue("InvoiceId")

	userID, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if err := h.subscriptionsService.CreateOrExtend(r.Context(), userID, invoiceID); err != nil {
		log.Print(err)
		http.Error(w, "failed to create a new subscription", http.StatusBadRequest)
		return
	}

	if err := h.bot.SendSuccessPayment(userID); err != nil {
		log.Print(err)
		http.Error(w, "failed to send success message", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"code":0}`))
}

func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	planID := query.Get("plan")
	userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	url, err := h.paymentsService.CreateInvoice(r.Context(), payments.CreateInvoiceInput{
		PlanID: planID,
		UserID: userID,
	})
	if err != nil {
		http.Error(w, "failed to create invoice", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
