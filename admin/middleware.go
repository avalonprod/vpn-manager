package admin

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type contextKey string

const claimsContextKey contextKey = "admin_claims"

// maxBodySize ограничивает размер тела запроса: админские payload'ы мелкие,
// а лимит закрывает тривиальный вектор исчерпания памяти.
const maxBodySize = 1 << 20 // 1 MiB

// ClaimsFrom достаёт claims авторизованного администратора из контекста.
func ClaimsFrom(ctx context.Context) (Claims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(Claims)
	return claims, ok
}

// authGuard пропускает только запросы с валидным Bearer-токеном.
//
// Токен принимается исключительно из заголовка Authorization — не из cookie,
// поэтому браузер не подставляет его автоматически и CSRF here невозможен.
func (h *Handler) authGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		token, found := strings.CutPrefix(header, "Bearer ")
		if !found || strings.TrimSpace(token) == "" {
			writeError(w, http.StatusUnauthorized, "authorization required")
			return
		}

		claims, err := h.tokens.Parse(strings.TrimSpace(token))
		if err != nil {
			if errors.Is(err, ErrTokenExpired) {
				writeError(w, http.StatusUnauthorized, "token expired")
				return
			}
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), claimsContextKey, claims)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// cors отвечает только тем Origin, которые перечислены в ADMIN_CORS_ORIGINS.
func (h *Handler) cors(next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(h.allowedOrigins))
	for _, origin := range h.allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimRight(r.Header.Get("Origin"), "/")

		if _, ok := allowed[origin]; ok && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Max-Age", "600")
			// Ответ зависит от Origin — иначе кэш отдаст чужие заголовки.
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// securityHeaders выставляет защитные заголовки и ограничивает тело запроса.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")

		r.Body = http.MaxBytesReader(w, r.Body, maxBodySize)

		next.ServeHTTP(w, r)
	})
}

// recoverPanic не даёт панике в обработчике уронить весь сервис и не
// возвращает наружу подробности ошибки.
func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				h.logger.Errorf("admin: panic on %s %s: %v", r.Method, r.URL.Path, rec)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// rateLimiter — token bucket на IP, общий для всех админских маршрутов.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
	rate    float64
	burst   float64
}

type bucket struct {
	tokens float64
	last   time.Time
}

func newRateLimiter(ratePerSecond, burst float64) *rateLimiter {
	return &rateLimiter{
		buckets: make(map[string]*bucket),
		rate:    ratePerSecond,
		burst:   burst,
	}
}

func (l *rateLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	b, ok := l.buckets[ip]
	if !ok {
		l.buckets[ip] = &bucket{tokens: l.burst - 1, last: now}
		l.evictLocked(now)
		return true
	}

	b.tokens = min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.rate)
	b.last = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--

	return true
}

// evictLocked чистит корзины, которые давно полны и ничего не ограничивают.
func (l *rateLimiter) evictLocked(now time.Time) {
	if len(l.buckets) < 1024 {
		return
	}

	for ip, b := range l.buckets {
		if now.Sub(b.last) > 10*time.Minute {
			delete(l.buckets, ip)
		}
	}
}

func (h *Handler) rateLimit(limiter *rateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !limiter.Allow(clientIP(r)) {
				writeError(w, http.StatusTooManyRequests, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// clientIP определяет адрес клиента. X-Forwarded-For учитывается только когда
// сервис явно запущен за доверенным прокси: иначе заголовок подделывается и
// обходит лимиты.
func clientIP(r *http.Request) string {
	if trustProxyHeaders {
		if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
			if first, _, ok := strings.Cut(forwarded, ","); ok {
				return strings.TrimSpace(first)
			}
			return strings.TrimSpace(forwarded)
		}

		if realIP := strings.TrimSpace(r.Header.Get("X-Real-Ip")); realIP != "" {
			return realIP
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// trustProxyHeaders включается переменной окружения ADMIN_TRUST_PROXY=true.
var trustProxyHeaders bool

// SetTrustProxyHeaders задаёт, доверять ли заголовкам прокси при определении IP.
func SetTrustProxyHeaders(trust bool) {
	trustProxyHeaders = trust
}
