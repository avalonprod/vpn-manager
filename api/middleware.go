package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"vpn-manager/subscriptions"
)

func (h *Handler) authorizeCloudPayment(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "authorizeCloudPayment"

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Error(err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(io.LimitReader(io.MultiReader(io.NopCloser(bytes.NewReader(bodyBytes))), int64(len(bodyBytes))))

		receivedHMAC := r.Header.Get("Content-HMAC")
		if receivedHMAC == "" {
			h.logger.Debug("Missing Content-HMAC header")
			http.Error(w, "Missing Content-HMAC header", http.StatusForbidden)
			return
		}

		mac := hmac.New(sha256.New, []byte(h.cloudPaymentsSecret))
		mac.Write(bodyBytes)
		expectedMAC := hex.EncodeToString(mac.Sum(nil))

		if !hmac.Equal([]byte(receivedHMAC), []byte(expectedMAC)) {
			h.logger.Error(err)
			http.Error(w, "Invalid HMAC signature", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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

		// Блокировка проверяется раньше подписки: заблокированный пользователь
		// не должен получить конфиги даже с оплаченной подпиской.
		blocked, err := h.usersService.IsBlocked(r.Context(), userID)
		if err != nil {
			h.logger.Errorf("%s: failed to check block status for user_id: %d error: %w", op, userID, err)
			http.Error(w, "failed to check access", http.StatusInternalServerError)
			return
		}

		if blocked {
			h.logger.Warnf("%s: blocked user_id: %d tried to access", op, userID)
			http.Error(w, "access denied", http.StatusForbidden)
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
