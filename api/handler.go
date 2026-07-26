package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"vpn-manager/core/config"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/pkg/logger"
	"vpn-manager/plans"
	"vpn-manager/tasks"

	"github.com/gorilla/mux"
)

type IPeersService interface {
	GetActivePeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
	GetByAccessToken(ctx context.Context, token string) (peers.Peer, error)
	SetImported(ctx context.Context, userID int64) error
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
	IsSubscriptionActive(ctx context.Context, userID int64) (bool, error)
}

type IUsersService interface {
	IsBlocked(ctx context.Context, userID int64) (bool, error)
}

type ITasksService interface {
	Enqueue(ctx context.Context, task tasks.Task) error
}

type IBot interface {
	SendSetupInstruction(userID int64, os string) error
	SendPostImportInstructions(userID int64, os string) error
	SendSuccessPayment(userID int64) error
}

type AdminRoutes interface {
	RegisterRoutes(router *mux.Router)
}

type Handler struct {
	peersService         IPeersService
	plansService         IPlansService
	paymentsService      IPaymentsService
	subscriptionsService ISubscriptionsService
	usersService         IUsersService
	tasksService         ITasksService
	bot                  IBot
	logger               logger.ILogger
	apiUrl               string
	cloudPaymentsSecret  string
	apps                 config.Apps
	admin                AdminRoutes
	allowLegacyLinks     bool
}

type Deps struct {
	Peers         IPeersService
	Plans         IPlansService
	Payments      IPaymentsService
	Subscriptions ISubscriptionsService
	Users         IUsersService
	Tasks         ITasksService
	Bot           IBot
	Logger        logger.ILogger
	ApiUrl        string
	CloudPayments string
	Apps          config.Apps

	Admin            AdminRoutes
	AllowLegacyLinks bool
}

func NewHandler(deps Deps) *Handler {
	return &Handler{
		peersService:         deps.Peers,
		plansService:         deps.Plans,
		paymentsService:      deps.Payments,
		subscriptionsService: deps.Subscriptions,
		usersService:         deps.Users,
		cloudPaymentsSecret:  deps.CloudPayments,
		tasksService:         deps.Tasks,
		bot:                  deps.Bot,
		logger:               deps.Logger,
		apiUrl:               deps.ApiUrl,
		apps:                 deps.Apps,
		admin:                deps.Admin,
		allowLegacyLinks:     deps.AllowLegacyLinks,
	}
}

func (h *Handler) RegisterRoutes() *mux.Router {
	r := mux.NewRouter()

	r.Handle("/subs", h.AccessGuard(http.HandlerFunc(h.getSubs))).Methods("GET")
	r.Handle("/setup", h.AccessGuard(http.HandlerFunc(h.setup))).Methods("GET")
	r.Handle("/apps", h.BlockGuard(http.HandlerFunc(h.downloadApp))).Methods("GET")
	r.Handle("/subscribe", h.BlockGuard(http.HandlerFunc(h.handleSubscribe))).Methods("GET")
	r.HandleFunc("/cloudpayments/webhook/check", h.handleCheckCloudPaymentsWebhook).Methods("GET")
	r.Handle("/cloudpayments/webhook/pay", h.authorizeCloudPayment(http.HandlerFunc(h.handlePayCloudPaymentsWebhook))).Methods("POST")

	if h.admin != nil {
		h.admin.RegisterRoutes(r)
	}

	return r
}

func (h *Handler) downloadApp(w http.ResponseWriter, r *http.Request) {
	const op = "downloadApp"

	query := r.URL.Query()

	userID, ok := userIDFrom(r.Context())
	if !ok {
		h.logger.Warnf("%s: request without an identified user", op)
		http.Error(w, "invalid link", http.StatusNotFound)
		return
	}

	if err := h.bot.SendSetupInstruction(userID, query.Get("os")); err != nil {
		h.logger.Errorf("%s: failed to send setup instruction for user_id: %v error: %w", op, userID, err)
		http.Error(w, "Failed to send setup instructions", http.StatusInternalServerError)
		return
	}

	if err := h.tasksService.Enqueue(r.Context(), tasks.Task{
		Type:        "setup_nudge",
		UserID:      userID,
		Payload:     []byte(query.Get("os")),
		RunAt:       time.Now().UTC().Add(5 * time.Minute),
		MaxAttempts: 10,
		DedupeKey:   fmt.Sprintf("setup_nudge:%d", userID),
	}); err != nil {
		h.logger.Errorf("%s: failed to send setup instruction for user_id: %v error: %w", op, userID, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
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

const profileName = "NeonGuard"

func (h *Handler) subscriptionURL(r *http.Request) string {
	query := r.URL.Query()

	if token := query.Get("token"); token != "" {
		return fmt.Sprintf("%s/subs?token=%s&name=%s", h.apiUrl, url.QueryEscape(token), profileName)
	}

	return fmt.Sprintf("%s/subs?user_id=%s&name=%s", h.apiUrl, url.QueryEscape(query.Get("user_id")), profileName)
}

func (h *Handler) setup(w http.ResponseWriter, r *http.Request) {
	const op = "setup"

	query := r.URL.Query()

	userID, ok := userIDFrom(r.Context())
	if !ok {
		h.logger.Warnf("%s: request without an identified user", op)
		http.Error(w, "invalid link", http.StatusNotFound)
		return
	}

	if err := h.bot.SendPostImportInstructions(userID, query.Get("os")); err != nil {
		h.logger.Warnf("%s: failed to send post import instruction for user_id: %v error: %w", op, userID, err)
		http.Error(w, "failed to send post import instructions", http.StatusInternalServerError)
		return
	}

	subsURL := h.subscriptionURL(r)

	var deep string

	switch query.Get("os") {
	case "ios", "macos":
		deep = fmt.Sprintf("streisand://import/%s", subsURL)
	case "android":
		deep = fmt.Sprintf("hiddify://import/%s#%s", subsURL, url.PathEscape(profileName))
	}

	http.Redirect(w, r, deep, http.StatusTemporaryRedirect)
}

func (h *Handler) getSubs(w http.ResponseWriter, r *http.Request) {
	const op = "getSubs"

	userID, ok := userIDFrom(r.Context())
	if !ok {
		h.logger.Warnf("%s: request without an identified user", op)
		http.Error(w, "invalid link", http.StatusNotFound)
		return
	}

	peer, err := h.peersService.GetActivePeerByUserID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, peers.ErrPeerNotFound) {
			h.logger.Debugf("%s: subs not found: %w", op, err)
			http.Error(w, "empty subs", http.StatusNotFound)
			return
		}
		h.logger.Errorf("%s: failed to get active peers for user_id: %s error: %w", op, userID, err)
		http.Error(w, "failed to get subs", http.StatusInternalServerError)
		return
	}

	subs := make([]string, 0, len(peer.Subs))

	for _, sub := range peer.Subs {
		subs = append(subs, sub.URL)
	}

	if err := h.peersService.SetImported(r.Context(), userID); err != nil {
		h.logger.Errorf("%s: failed to set imported for user_id: %d error: %w", op, userID, err)
		http.Error(w, "failed to set imported", http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, strings.Join(subs, "\n"))
}

func (h *Handler) handleCheckCloudPaymentsWebhook(w http.ResponseWriter, r *http.Request) {
	const op = "handleCheckCloudPaymentsWebhook"

	w.Header().Set("Content-Type", "application/json")

	query := r.URL.Query()

	accountId := query.Get("AccountId")
	invoiceId := query.Get("InvoiceId")

	userID, err := strconv.ParseInt(accountId, 10, 64)
	if err != nil {
		h.logger.Warnf("%s: invalid user_id: %v error: %w", op, userID, err)
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	_, err = h.paymentsService.GetInvoiceByID(r.Context(), userID, invoiceId)
	if err != nil {
		h.logger.Errorf("%s: failed to find invoice with id: %s for user_id: %s error: %w", op, invoiceId, userID, err)
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
	const op = "handlePayCloudPaymentsWebhook"
	if err := r.ParseForm(); err != nil {
		h.logger.Error("%s: failed to parse form: %w", op, err)
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	accountID := r.FormValue("AccountId")
	invoiceID := r.FormValue("InvoiceId")

	userID, err := strconv.ParseInt(accountID, 10, 64)
	if err != nil {
		h.logger.Warnf("%s: invalid user_id: %v error: %w", op, userID, err)
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	if err := h.subscriptionsService.CreateOrExtend(r.Context(), userID, invoiceID); err != nil {
		h.logger.Errorf("%s: failed to create subscription for user_id: %d error: %w", op, userID, err)
		http.Error(w, "failed to create a new subscription", http.StatusBadRequest)
		return
	}

	if err := h.bot.SendSuccessPayment(userID); err != nil {
		h.logger.Errorf("%s: failed to send success message for user_id: %d error: %w", op, userID, err)
		http.Error(w, "failed to send success message", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"code":0}`))
}

func (h *Handler) handleSubscribe(w http.ResponseWriter, r *http.Request) {
	const op = "handleSubscribe"
	query := r.URL.Query()

	planID := query.Get("plan")
	userID, ok := userIDFrom(r.Context())
	if !ok {
		h.logger.Warnf("%s: request without an identified user", op)
		http.Error(w, "invalid link", http.StatusNotFound)
		return
	}

	url, err := h.paymentsService.CreateInvoice(r.Context(), payments.CreateInvoiceInput{
		PlanID: planID,
		UserID: userID,
	})
	if err != nil {
		h.logger.Errorf("%s: failed to create invoice user_id: %d error: %w", op, userID, err)
		http.Error(w, "failed to create invoice", http.StatusBadGateway)
		return
	}

	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}
