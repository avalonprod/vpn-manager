package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strconv"
	"vpn-manager/peers"
	"vpn-manager/subscriptions"
)

const maxWebhookBody = 1 << 20

func (h *Handler) authorizeCloudPayment(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "authorizeCloudPayment"

		bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, maxWebhookBody))
		if err != nil {
			h.logger.Errorf("%s: failed to read body: %v", op, err)
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

		received := r.Header.Get("Content-HMAC")
		if received == "" {
			received = r.Header.Get("X-Content-HMAC")
		}

		if received == "" {
			h.logger.Warnf("%s: request without an HMAC header", op)
			http.Error(w, "Missing Content-HMAC header", http.StatusForbidden)
			return
		}

		signature, err := base64.StdEncoding.DecodeString(received)
		if err != nil {
			h.logger.Warnf("%s: HMAC header is not valid base64", op)
			http.Error(w, "Invalid HMAC signature", http.StatusForbidden)
			return
		}

		mac := hmac.New(sha256.New, []byte(h.cloudPaymentsSecret))
		mac.Write(bodyBytes)

		if !hmac.Equal(signature, mac.Sum(nil)) {
			h.logger.Warnf("%s: HMAC signature mismatch", op)
			http.Error(w, "Invalid HMAC signature", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type contextKey string

const userIDContextKey contextKey = "user_id"

func userIDFrom(ctx context.Context) (int64, bool) {
	userID, ok := ctx.Value(userIDContextKey).(int64)
	return userID, ok
}

func (h *Handler) Identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "Identify"

		query := r.URL.Query()

		if token := query.Get("token"); token != "" {
			peer, err := h.peersService.GetByAccessToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, peers.ErrPeerNotFound) {
					h.logger.Warnf("%s: unknown access token", op)
					http.Error(w, "invalid link", http.StatusNotFound)
					return
				}
				h.logger.Errorf("%s: failed to resolve access token: %v", op, err)
				http.Error(w, "failed to resolve link", http.StatusInternalServerError)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey, peer.UserID)))
			return
		}

		if !h.allowLegacyLinks {
			h.logger.Warnf("%s: request without an access token to %s", op, r.URL.Path)
			http.Error(w, "invalid link", http.StatusNotFound)
			return
		}

		userID, err := strconv.ParseInt(query.Get("user_id"), 10, 64)
		if err != nil {
			h.logger.Warnf("%s: invalid user_id: %v", op, err)
			http.Error(w, "invalid user_id", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userIDContextKey, userID)))
	})
}

func (h *Handler) BlockGuard(next http.Handler) http.Handler {
	return h.Identify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "BlockGuard"

		userID, ok := userIDFrom(r.Context())
		if !ok {
			http.Error(w, "invalid link", http.StatusNotFound)
			return
		}

		blocked, err := h.usersService.IsBlocked(r.Context(), userID)
		if err != nil {
			h.logger.Errorf("%s: failed to check block status for user_id: %d error: %w", op, userID, err)
			http.Error(w, "failed to check access", http.StatusInternalServerError)
			return
		}

		if blocked {
			h.logger.Warnf("%s: blocked user_id: %d tried to access %s", op, userID, r.URL.Path)
			http.Error(w, "access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	}))
}

func (h *Handler) AccessGuard(next http.Handler) http.Handler {
	return h.BlockGuard(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const op = "AccessGuard"

		userID, ok := userIDFrom(r.Context())
		if !ok {
			h.logger.Warnf("%s: request without an identified user", op)
			http.Error(w, "invalid link", http.StatusNotFound)
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
	}))
}
