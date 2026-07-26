package admin

import (
	"context"
	"net/http"
	"time"
	"vpn-manager/core/config"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/pkg/logger"
	"vpn-manager/plans"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"github.com/gorilla/mux"
)

type IUsersService interface {
	GetByID(ctx context.Context, ID int64) (users.User, error)
	List(ctx context.Context, f users.ListFilter) ([]users.User, int64, error)
	SetBlocked(ctx context.Context, userID int64, blocked bool, reason string) error
	Totals(ctx context.Context) (users.Totals, error)
	SignupsByDay(ctx context.Context, since time.Time) ([]users.DailyCount, error)
}

type IServersService interface {
	GetAll(ctx context.Context) ([]servers.Server, error)
	GetByID(ctx context.Context, serverID string) (servers.Server, error)
	Create(ctx context.Context, input servers.CreateInput) (servers.Server, error)
	Update(ctx context.Context, serverID string, input servers.UpdateInput) (servers.Server, error)
	Delete(ctx context.Context, serverID string) error
	Count(ctx context.Context) (total int64, active int64, err error)
	CheckHealth(ctx context.Context, serverID string) (servers.Health, error)
	CheckAllHealth(ctx context.Context) ([]servers.Health, error)
	DeletePeerFromServer(ctx context.Context, serverID, email string) error
	RevokeAccessEverywhere(ctx context.Context, email string) (servers.RevocationResult, error)
	RegisterNewPeers(ctx context.Context, userID int64) error
}

type IPlansService interface {
	GetAllIncludingInactive(ctx context.Context) ([]plans.Plan, error)
	GetByID(ctx context.Context, ID string) (plans.Plan, error)
	Create(ctx context.Context, input plans.CreateInput) (plans.Plan, error)
	Update(ctx context.Context, ID string, input plans.UpdateInput) (plans.Plan, error)
	Delete(ctx context.Context, ID string) error
}

type ISubscriptionsService interface {
	GetByUserID(ctx context.Context, userID int64) (subscriptions.Subscription, error)
	GetSubscriptionsForUsers(ctx context.Context, userIDs []int64) (map[int64]subscriptions.Subscription, error)
	Totals(ctx context.Context) (subscriptions.Totals, error)
	CountByPlan(ctx context.Context) ([]subscriptions.PlanCount, error)
	CreatedByDay(ctx context.Context, since time.Time, trialOnly *bool) ([]subscriptions.DailyCount, error)
	Deactivate(ctx context.Context, userID int64) error
}

type IPaymentsService interface {
	List(ctx context.Context, f payments.ListFilter) ([]payments.Invoice, int64, error)
	GetByUserID(ctx context.Context, userID int64, limit int) ([]payments.Invoice, error)
	Totals(ctx context.Context) (payments.Totals, error)
	RevenueSince(ctx context.Context, since time.Time) (float64, error)
	RevenueByDay(ctx context.Context, since time.Time) ([]payments.DailyRevenue, error)
	RevenueByPlan(ctx context.Context, since time.Time) ([]payments.PlanRevenue, error)
}

type IPeersService interface {
	GetPeerByUserID(ctx context.Context, userID int64) (peers.Peer, error)
	GetPeersForUsers(ctx context.Context, userIDs []int64) (map[int64]peers.Peer, error)
	ActivatePeer(ctx context.Context, userID int64) error
	DeactivatePeer(ctx context.Context, userID int64) error
	Totals(ctx context.Context) (peers.Totals, error)
	CountByLocation(ctx context.Context) ([]peers.LocationCount, error)
}

type Handler struct {
	usersService         IUsersService
	serversService       IServersService
	plansService         IPlansService
	subscriptionsService ISubscriptionsService
	paymentsService      IPaymentsService
	peersService         IPeersService
	auditStore           *AuditStore
	logger               logger.ILogger

	auth           *authenticator
	tokens         *tokenIssuer
	allowedOrigins []string
}

type Deps struct {
	Users         IUsersService
	Servers       IServersService
	Plans         IPlansService
	Subscriptions ISubscriptionsService
	Payments      IPaymentsService
	Peers         IPeersService
	Audit         *AuditStore
	Logger        logger.ILogger
}

func NewHandler(cfg config.Admin, deps Deps) *Handler {
	return &Handler{
		usersService:         deps.Users,
		serversService:       deps.Servers,
		plansService:         deps.Plans,
		subscriptionsService: deps.Subscriptions,
		paymentsService:      deps.Payments,
		peersService:         deps.Peers,
		auditStore:           deps.Audit,
		logger:               deps.Logger,
		auth:                 newAuthenticator(cfg),
		tokens:               newTokenIssuer(cfg.JWTSecret, cfg.TokenTTL),
		allowedOrigins:       cfg.AllowedOrigins,
	}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	api := router.PathPrefix("/admin/api/v1").Subrouter()
	api.Use(h.recoverPanic, securityHeaders, h.cors)

	login := api.Path("/auth/login").Subrouter()
	login.Use(h.rateLimit(newRateLimiter(0.2, 10)))
	login.HandleFunc("", h.handleLogin).Methods(http.MethodPost, http.MethodOptions)

	protected := api.PathPrefix("").Subrouter()
	protected.Use(h.rateLimit(newRateLimiter(20, 60)), h.authGuard)

	protected.HandleFunc("/auth/me", h.handleMe).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/auth/refresh", h.handleRefresh).Methods(http.MethodPost, http.MethodOptions)

	protected.HandleFunc("/analytics/overview", h.handleOverview).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/analytics/timeseries", h.handleTimeseries).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/analytics/breakdown", h.handleBreakdown).Methods(http.MethodGet, http.MethodOptions)

	protected.HandleFunc("/users", h.handleListUsers).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/users/{id}", h.handleGetUser).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/users/{id}/block", h.handleBlockUser).Methods(http.MethodPost, http.MethodOptions)
	protected.HandleFunc("/users/{id}/unblock", h.handleUnblockUser).Methods(http.MethodPost, http.MethodOptions)

	protected.HandleFunc("/servers/health", h.handleServersHealth).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/servers", h.handleListServers).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/servers", h.handleCreateServer).Methods(http.MethodPost, http.MethodOptions)
	protected.HandleFunc("/servers/{id}", h.handleUpdateServer).Methods(http.MethodPatch, http.MethodOptions)
	protected.HandleFunc("/servers/{id}", h.handleDeleteServer).Methods(http.MethodDelete, http.MethodOptions)
	protected.HandleFunc("/servers/{id}/health", h.handleServerHealth).Methods(http.MethodGet, http.MethodOptions)

	protected.HandleFunc("/plans", h.handleListPlans).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/plans", h.handleCreatePlan).Methods(http.MethodPost, http.MethodOptions)
	protected.HandleFunc("/plans/{id}", h.handleUpdatePlan).Methods(http.MethodPatch, http.MethodOptions)
	protected.HandleFunc("/plans/{id}", h.handleDeletePlan).Methods(http.MethodDelete, http.MethodOptions)

	protected.HandleFunc("/payments", h.handleListPayments).Methods(http.MethodGet, http.MethodOptions)
	protected.HandleFunc("/audit", h.handleListAudit).Methods(http.MethodGet, http.MethodOptions)
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
	Username  string `json:"username"`
}

func (h *Handler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	ip := clientIP(r)

	if err := h.auth.Authenticate(ip, req.Username, req.Password); err != nil {
		switch err {
		case ErrTooManyAttempts:
			h.logger.Warnf("admin: login locked out for %s", ip)
			writeError(w, http.StatusTooManyRequests, "too many login attempts, try again later")
		default:
			h.logger.Warnf("admin: failed login attempt from %s", ip)

			writeError(w, http.StatusUnauthorized, "invalid username or password")
		}
		return
	}

	token, claims, err := h.tokens.Issue(req.Username)
	if err != nil {
		h.logger.Errorf("admin: failed to issue token: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	h.audit(r.Context(), req.Username, ip, "auth.login", "", "")

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: claims.ExpiresAt,
		Username:  claims.Subject,
	})
}

func (h *Handler) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"username":   claims.Subject,
		"expires_at": claims.ExpiresAt,
	})
}

func (h *Handler) handleRefresh(w http.ResponseWriter, r *http.Request) {
	claims, ok := ClaimsFrom(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "authorization required")
		return
	}

	token, newClaims, err := h.tokens.Issue(claims.Subject)
	if err != nil {
		h.logger.Errorf("admin: failed to refresh token: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: newClaims.ExpiresAt,
		Username:  newClaims.Subject,
	})
}

func (h *Handler) handleListAudit(w http.ResponseWriter, r *http.Request) {
	if h.auditStore == nil {
		writeJSON(w, http.StatusOK, page[AuditEntry]{Items: []AuditEntry{}})
		return
	}

	limit, offset := pagination(r)

	entries, total, err := h.auditStore.List(r.Context(), limit, offset)
	if err != nil {
		h.logger.Errorf("admin: failed to list audit entries: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load audit log")
		return
	}

	writeJSON(w, http.StatusOK, page[AuditEntry]{
		Items:  entries,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}
