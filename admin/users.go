package admin

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
	"vpn-manager/payments"
	"vpn-manager/peers"
	"vpn-manager/servers"
	"vpn-manager/subscriptions"
	"vpn-manager/users"

	"github.com/gorilla/mux"
)

type userSummary struct {
	ID           int64      `json:"id"`
	Username     string     `json:"username"`
	FirstName    string     `json:"first_name"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
	IsBlocked    bool       `json:"is_blocked"`
	BlockReason  string     `json:"block_reason,omitempty"`
	BlockedAt    *time.Time `json:"blocked_at,omitempty"`

	SubscriptionActive bool       `json:"subscription_active"`
	SubscriptionTrial  bool       `json:"subscription_trial"`
	PlanID             string     `json:"plan_id,omitempty"`
	ExpiresAt          *time.Time `json:"expires_at,omitempty"`
	PeerActive         bool       `json:"peer_active"`
	PeerImported       bool       `json:"peer_imported"`
}

func toUserSummary(user users.User, sub subscriptions.Subscription, hasSub bool, peer peers.Peer, hasPeer bool) userSummary {
	summary := userSummary{
		ID:           user.ID,
		Username:     user.Username,
		FirstName:    user.FirstName,
		CreatedAt:    user.CreatedAt,
		LastActiveAt: user.LastActiveAt,
		IsBlocked:    user.IsBlocked,
		BlockReason:  user.BlockReason,
	}

	if !user.BlockedAt.IsZero() {
		blockedAt := user.BlockedAt
		summary.BlockedAt = &blockedAt
	}

	if hasSub {
		summary.SubscriptionActive = sub.Active
		summary.SubscriptionTrial = sub.IsTrial
		summary.PlanID = sub.PlanID
		expiresAt := sub.ExpiresAt
		summary.ExpiresAt = &expiresAt
	}

	if hasPeer {
		summary.PeerActive = peer.IsActive
		summary.PeerImported = peer.IsImported
	}

	return summary
}

func (h *Handler) handleListUsers(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	limit, offset := pagination(r)

	filter := users.ListFilter{
		Search:    strings.TrimSpace(query.Get("search")),
		SortField: query.Get("sort"),
		SortAsc:   query.Get("order") == "asc",
		Limit:     limit,
		Offset:    offset,
	}

	switch query.Get("status") {
	case "blocked":
		filter.Blocked = users.BlockedOnly
	case "active":
		filter.Blocked = users.BlockedExclude
	}

	list, total, err := h.usersService.List(r.Context(), filter)
	if err != nil {
		h.logger.Errorf("admin: failed to list users: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	ids := make([]int64, 0, len(list))
	for _, user := range list {
		ids = append(ids, user.ID)
	}

	subsByUser, err := h.subscriptionsService.GetSubscriptionsForUsers(r.Context(), ids)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscriptions for users: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	peersByUser, err := h.peersService.GetPeersForUsers(r.Context(), ids)
	if err != nil {
		h.logger.Errorf("admin: failed to load peers for users: %v", err)
		writeError(w, http.StatusInternalServerError, "failed to load users")
		return
	}

	items := make([]userSummary, 0, len(list))
	for _, user := range list {
		sub, hasSub := subsByUser[user.ID]
		peer, hasPeer := peersByUser[user.ID]
		items = append(items, toUserSummary(user, sub, hasSub, peer, hasPeer))
	}

	writeJSON(w, http.StatusOK, page[userSummary]{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

type userConnection struct {
	Location string `json:"location"`
	ServerID string `json:"server_id"`
	Enabled  bool   `json:"enabled"`
}

type userInvoice struct {
	ID        string    `json:"id"`
	PlanID    string    `json:"plan_id"`
	PlanTitle string    `json:"plan_title,omitempty"`
	Price     float64   `json:"price"`
	Currency  string    `json:"currency,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

type userDetails struct {
	userSummary
	AutoRenewal    bool             `json:"auto_renewal"`
	PlanTitle      string           `json:"plan_title,omitempty"`
	PeerUUID       string           `json:"peer_uuid,omitempty"`
	PeerCreatedAt  *time.Time       `json:"peer_created_at,omitempty"`
	PeerImportedAt *time.Time       `json:"peer_imported_at,omitempty"`
	Connections    []userConnection `json:"connections"`
	Invoices       []userInvoice    `json:"invoices"`
	TotalPaid      float64          `json:"total_paid"`
	PaidInvoices   int              `json:"paid_invoices"`
	DaysWithUs     int              `json:"days_with_us"`
}

const userInvoicesLimit = 50

func (h *Handler) handleGetUser(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDParam(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	ctx := r.Context()

	user, err := h.usersService.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Errorf("admin: failed to load user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	sub, hasSub, err := h.lookupSubscription(ctx, userID)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscription for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	peer, hasPeer, err := h.lookupPeer(ctx, userID)
	if err != nil {
		h.logger.Errorf("admin: failed to load peer for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	details := userDetails{
		userSummary: toUserSummary(user, sub, hasSub, peer, hasPeer),
		Connections: []userConnection{},
		Invoices:    []userInvoice{},
	}

	if hasSub {
		details.AutoRenewal = sub.AutoRenewal
		if plan, err := h.plansService.GetByID(ctx, sub.PlanID); err == nil {
			details.PlanTitle = plan.Title
		}
	}

	if hasPeer {
		details.PeerUUID = peer.UUID
		if !peer.CreatedAt.IsZero() {
			createdAt := peer.CreatedAt
			details.PeerCreatedAt = &createdAt
		}
		if !peer.ImportedAt.IsZero() {
			importedAt := peer.ImportedAt
			details.PeerImportedAt = &importedAt
		}
		for _, s := range peer.Subs {
			details.Connections = append(details.Connections, userConnection{
				Location: s.Location,
				ServerID: s.ServerID,
				Enabled:  s.Enabled,
			})
		}
	}

	invoices, err := h.paymentsService.GetByUserID(ctx, userID, userInvoicesLimit)
	if err != nil {
		h.logger.Errorf("admin: failed to load invoices for user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to load user")
		return
	}

	planCache := make(map[string]struct {
		title    string
		price    float64
		currency string
	})

	for _, invoice := range invoices {
		info, cached := planCache[invoice.PlanID]
		if !cached {
			if plan, err := h.plansService.GetByID(ctx, invoice.PlanID); err == nil {
				info.title, info.price, info.currency = plan.Title, plan.Price, plan.Currency
			}
			planCache[invoice.PlanID] = info
		}

		details.Invoices = append(details.Invoices, userInvoice{
			ID:        invoice.ID,
			PlanID:    invoice.PlanID,
			PlanTitle: info.title,
			Price:     info.price,
			Currency:  info.currency,
			Status:    invoice.Status,
			CreatedAt: invoice.CreatedAt,
		})

		if invoice.Status == payments.StatusCompleted {
			details.TotalPaid += info.price
			details.PaidInvoices++
		}
	}

	if !user.CreatedAt.IsZero() {
		details.DaysWithUs = int(time.Since(user.CreatedAt).Hours() / 24)
	}

	writeJSON(w, http.StatusOK, details)
}

func (h *Handler) lookupSubscription(ctx context.Context, userID int64) (subscriptions.Subscription, bool, error) {
	sub, err := h.subscriptionsService.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, subscriptions.ErrSubscriptionNotFound) {
			return subscriptions.Subscription{}, false, nil
		}
		return subscriptions.Subscription{}, false, err
	}

	return sub, true, nil
}

func (h *Handler) lookupPeer(ctx context.Context, userID int64) (peers.Peer, bool, error) {
	peer, err := h.peersService.GetPeerByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, peers.ErrPeerNotFound) {
			return peers.Peer{}, false, nil
		}
		return peers.Peer{}, false, err
	}

	return peer, true, nil
}

type blockRequest struct {
	Reason string `json:"reason"`
}

const maxBlockReasonLen = 500

func (h *Handler) handleBlockUser(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDParam(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var req blockRequest
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
	}

	reason := strings.TrimSpace(req.Reason)
	if len(reason) > maxBlockReasonLen {
		reason = reason[:maxBlockReasonLen]
	}

	ctx := r.Context()

	if err := h.usersService.SetBlocked(ctx, userID, true, reason); err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Errorf("admin: failed to block user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to block user")
		return
	}

	revoked, failed := h.revokeAccess(ctx, userID)

	claims, _ := ClaimsFrom(ctx)
	h.audit(ctx, claims.Subject, clientIP(r), "user.block", strconv.FormatInt(userID, 10), reason)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":            "blocked",
		"revoked_servers":   revoked,
		"failed_revocation": failed,
	})
}

func (h *Handler) revokeAccess(ctx context.Context, userID int64) (revoked int, failed []string) {
	failed = []string{}

	peer, hasPeer, err := h.lookupPeer(ctx, userID)
	if err != nil {
		h.logger.Errorf("admin: failed to load peer for user %d: %v", userID, err)
		return 0, []string{"не удалось прочитать подключения пользователя"}
	}

	if !hasPeer {
		return 0, failed
	}

	result, err := h.serversService.RevokeAccessEverywhere(ctx, peer.Email)
	if err != nil {
		h.logger.Errorf("admin: failed to revoke access for user %d: %v", userID, err)
		return 0, []string{"не удалось получить список серверов"}
	}

	revoked, failed = result.Revoked, result.Failed

	if err := h.peersService.DeactivatePeer(ctx, userID); err != nil {
		h.logger.Errorf("admin: failed to deactivate peer for user %d: %v", userID, err)
	}

	return revoked, failed
}

func (h *Handler) handleUnblockUser(w http.ResponseWriter, r *http.Request) {
	userID, err := userIDParam(mux.Vars(r)["id"])
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	ctx := r.Context()

	if err := h.usersService.SetBlocked(ctx, userID, false, ""); err != nil {
		if errors.Is(err, users.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		h.logger.Errorf("admin: failed to unblock user %d: %v", userID, err)
		writeError(w, http.StatusInternalServerError, "failed to unblock user")
		return
	}

	accessRestored, reason := h.restoreAccess(ctx, userID)

	claims, _ := ClaimsFrom(ctx)
	h.audit(ctx, claims.Subject, clientIP(r), "user.unblock", strconv.FormatInt(userID, 10), reason)

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "active",
		"access_restored": accessRestored,
		"reason":          reason,
	})
}

func (h *Handler) restoreAccess(ctx context.Context, userID int64) (bool, string) {
	sub, hasSub, err := h.lookupSubscription(ctx, userID)
	if err != nil {
		h.logger.Errorf("admin: failed to load subscription for user %d: %v", userID, err)
		return false, "не удалось прочитать подписку"
	}

	if !hasSub {
		return false, "у пользователя нет подписки"
	}

	if !sub.Active || !sub.ExpiresAt.After(time.Now().UTC()) {
		return false, "подписка неактивна или истекла"
	}

	if err := h.serversService.RegisterNewPeers(ctx, userID); err != nil {
		h.logger.Errorf("admin: failed to re-register peers for user %d: %v", userID, err)

		if errors.Is(err, servers.ErrNoServersAvailable) {
			return false, "ни один сервер не принял клиента — проверьте доступность панелей"
		}

		return false, "не удалось вернуть подключения на серверы"
	}

	if err := h.peersService.ActivatePeer(ctx, userID); err != nil {
		h.logger.Errorf("admin: failed to activate peer for user %d: %v", userID, err)
		return false, "подключения выданы, но пир не активирован"
	}

	return true, ""
}
