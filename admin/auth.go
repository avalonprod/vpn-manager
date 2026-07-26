package admin

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"
	"sync"
	"time"
	"vpn-manager/core/config"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTooManyAttempts    = errors.New("too many login attempts")
)

const (
	maxLoginAttempts = 5
	loginWindow      = 15 * time.Minute
	lockoutDuration  = 15 * time.Minute
)

type authenticator struct {
	username     []byte
	passwordHash []byte
	limiter      *loginLimiter
}

func newAuthenticator(cfg config.Admin) *authenticator {

	passwordHash := sha256.Sum256([]byte(cfg.Password))
	hash := passwordHash[:]

	if cfg.PasswordSHA256 != "" {
		if decoded, err := hex.DecodeString(cfg.PasswordSHA256); err == nil && len(decoded) == sha256.Size {
			hash = decoded
		}
	}

	usernameHash := sha256.Sum256([]byte(cfg.Username))

	return &authenticator{
		username:     usernameHash[:],
		passwordHash: hash,
		limiter:      newLoginLimiter(),
	}
}

func (a *authenticator) Authenticate(ip, username, password string) error {
	if !a.limiter.Allow(ip) {
		return ErrTooManyAttempts
	}

	givenUsername := sha256.Sum256([]byte(strings.TrimSpace(username)))
	givenPassword := sha256.Sum256([]byte(password))

	usernameOK := subtle.ConstantTimeCompare(a.username, givenUsername[:])
	passwordOK := subtle.ConstantTimeCompare(a.passwordHash, givenPassword[:])

	if usernameOK&passwordOK != 1 {
		a.limiter.Fail(ip)
		return ErrInvalidCredentials
	}

	a.limiter.Reset(ip)

	return nil
}

type attemptState struct {
	failures    int
	windowStart time.Time
	lockedUntil time.Time
}

type loginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*attemptState
}

func newLoginLimiter() *loginLimiter {
	return &loginLimiter{attempts: make(map[string]*attemptState)}
}

func (l *loginLimiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.evictLocked()

	state, ok := l.attempts[ip]
	if !ok {
		return true
	}

	return time.Now().UTC().After(state.lockedUntil)
}

func (l *loginLimiter) Fail(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()

	state, ok := l.attempts[ip]
	if !ok || now.Sub(state.windowStart) > loginWindow {
		l.attempts[ip] = &attemptState{failures: 1, windowStart: now}
		return
	}

	state.failures++
	if state.failures >= maxLoginAttempts {
		state.lockedUntil = now.Add(lockoutDuration)
		state.failures = 0
		state.windowStart = now
	}
}

func (l *loginLimiter) Reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.attempts, ip)
}

func (l *loginLimiter) evictLocked() {
	now := time.Now().UTC()

	for ip, state := range l.attempts {
		if now.After(state.lockedUntil) && now.Sub(state.windowStart) > loginWindow {
			delete(l.attempts, ip)
		}
	}
}
