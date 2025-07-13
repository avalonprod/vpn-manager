package api

import (
	"errors"
	"net/http"
	"strconv"
	"vpn-manager/subscriptions"
)

func (h *Handler) AccessGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "AccessGuard"

		query := r.URL.Query()

		userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
		if err != nil {
			h.logger.Warnf("%s: invalid user_id: %v error: %w", op, userID, err)
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}
		isActive, err := h.subscriptionsService.IsSubscriptionActive(r.Context(), userID)
		if err != nil {
			if errors.Is(err, subscriptions.ErrSubscriptionNotFound) {
				http.Error(w, "your subscription is not found", http.StatusNotFound)
				return
			}
			h.logger.Warnf("%s: failed to check subscription status for user_id: %s error: %w", op, userID, err)
			http.Error(w, "failed to check subscription status", http.StatusInternalServerError)
			return
		}

		if !isActive {
			http.Error(w, "your subscription not active", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}
