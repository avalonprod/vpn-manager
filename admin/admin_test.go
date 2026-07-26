package admin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"
	"vpn-manager/core/config"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestIssuedTokenParsesBack(t *testing.T) {
	issuer := newTokenIssuer(testSecret, time.Hour)

	token, claims, err := issuer.Issue("admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	parsed, err := issuer.Parse(token)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if parsed.Subject != "admin" {
		t.Errorf("subject = %q, want %q", parsed.Subject, "admin")
	}

	if parsed.ExpiresAt != claims.ExpiresAt {
		t.Errorf("exp = %d, want %d", parsed.ExpiresAt, claims.ExpiresAt)
	}

	if parsed.TokenID == "" {
		t.Error("jti is empty")
	}
}

func TestParseRejectsTamperedPayload(t *testing.T) {
	issuer := newTokenIssuer(testSecret, time.Hour)

	token, _, err := issuer.Issue("admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	parts := strings.Split(token, ".")

	forged, err := encodeSegment(Claims{
		Subject:   "attacker",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if _, err := issuer.Parse(parts[0] + "." + forged + "." + parts[2]); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

// Токен, подписанный чужим ключом, не должен приниматься.
func TestParseRejectsForeignSignature(t *testing.T) {
	token, _, err := newTokenIssuer("ffffffffffffffffffffffffffffffff", time.Hour).Issue("admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := newTokenIssuer(testSecret, time.Hour).Parse(token); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

// Классическая атака alg=none должна отбиваться: подпись проверяется всегда.
func TestParseRejectsAlgNone(t *testing.T) {
	header, err := encodeSegment(jwtHeader{Algorithm: "none", Type: "JWT"})
	if err != nil {
		t.Fatalf("encode header: %v", err)
	}

	payload, err := encodeSegment(Claims{
		Subject:   "attacker",
		ExpiresAt: time.Now().Add(time.Hour).Unix(),
	})
	if err != nil {
		t.Fatalf("encode payload: %v", err)
	}

	if _, err := newTokenIssuer(testSecret, time.Hour).Parse(header + "." + payload + "."); err != ErrInvalidToken {
		t.Errorf("err = %v, want ErrInvalidToken", err)
	}
}

func TestParseRejectsExpiredToken(t *testing.T) {
	issuer := newTokenIssuer(testSecret, -time.Minute)

	token, _, err := issuer.Issue("admin")
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := issuer.Parse(token); err != ErrTokenExpired {
		t.Errorf("err = %v, want ErrTokenExpired", err)
	}
}

func TestParseRejectsMalformedToken(t *testing.T) {
	issuer := newTokenIssuer(testSecret, time.Hour)

	// Валидно подписанный, но не-JSON payload не должен проходить.
	header, _ := encodeSegment(jwtHeader{Algorithm: "HS256", Type: "JWT"})
	garbage := base64.RawURLEncoding.EncodeToString([]byte("not json"))
	signed := header + "." + garbage
	signed += "." + issuer.sign(signed)

	for _, token := range []string{"", "a.b", "a.b.c.d", signed} {
		if _, err := issuer.Parse(token); err == nil {
			t.Errorf("Parse(%q) = nil error, want error", token)
		}
	}
}

func TestAuthenticateAcceptsOnlyExactCredentials(t *testing.T) {
	auth := newAuthenticator(config.Admin{Username: "root", Password: "s3cret"})

	if err := auth.Authenticate("1.1.1.1", "root", "s3cret"); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}

	cases := []struct{ username, password string }{
		{"root", "wrong"},
		{"admin", "s3cret"},
		{"", ""},
		{"root", ""},
	}

	for _, c := range cases {
		if err := auth.Authenticate("2.2.2.2", c.username, c.password); err != ErrInvalidCredentials {
			t.Errorf("Authenticate(%q, %q) = %v, want ErrInvalidCredentials", c.username, c.password, err)
		}
	}
}

func TestAuthenticateAcceptsPasswordHash(t *testing.T) {
	digest := sha256.Sum256([]byte("s3cret"))

	auth := newAuthenticator(config.Admin{
		Username:       "root",
		PasswordSHA256: hex.EncodeToString(digest[:]),
	})

	if err := auth.Authenticate("1.1.1.1", "root", "s3cret"); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}

	if err := auth.Authenticate("1.1.1.1", "root", "nope"); err != ErrInvalidCredentials {
		t.Errorf("err = %v, want ErrInvalidCredentials", err)
	}
}

func TestAuthenticateLocksOutAfterRepeatedFailures(t *testing.T) {
	auth := newAuthenticator(config.Admin{Username: "root", Password: "s3cret"})

	for range maxLoginAttempts {
		if err := auth.Authenticate("3.3.3.3", "root", "wrong"); err != ErrInvalidCredentials {
			t.Fatalf("err = %v, want ErrInvalidCredentials", err)
		}
	}

	// После исчерпания попыток блокируется даже верный пароль.
	if err := auth.Authenticate("3.3.3.3", "root", "s3cret"); err != ErrTooManyAttempts {
		t.Errorf("err = %v, want ErrTooManyAttempts", err)
	}

	// Лимит привязан к IP: другой адрес не должен страдать.
	if err := auth.Authenticate("4.4.4.4", "root", "s3cret"); err != nil {
		t.Errorf("unrelated ip blocked: %v", err)
	}
}

func TestSuccessfulLoginResetsFailureCounter(t *testing.T) {
	auth := newAuthenticator(config.Admin{Username: "root", Password: "s3cret"})

	for range maxLoginAttempts - 1 {
		_ = auth.Authenticate("5.5.5.5", "root", "wrong")
	}

	if err := auth.Authenticate("5.5.5.5", "root", "s3cret"); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}

	for range maxLoginAttempts - 1 {
		if err := auth.Authenticate("5.5.5.5", "root", "wrong"); err != ErrInvalidCredentials {
			t.Fatalf("counter was not reset: %v", err)
		}
	}
}

func TestRateLimiterAllowsBurstThenBlocks(t *testing.T) {
	limiter := newRateLimiter(0.0001, 3)

	for i := range 3 {
		if !limiter.Allow("9.9.9.9") {
			t.Fatalf("request %d blocked within burst", i)
		}
	}

	if limiter.Allow("9.9.9.9") {
		t.Error("request past burst was allowed")
	}
}

// Claims сериализуются в стандартные для JWT имена полей.
func TestClaimsUseStandardJSONNames(t *testing.T) {
	raw, err := json.Marshal(Claims{Subject: "admin", IssuedAt: 1, ExpiresAt: 2, TokenID: "x"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	for _, field := range []string{`"sub"`, `"iat"`, `"exp"`, `"jti"`} {
		if !strings.Contains(string(raw), field) {
			t.Errorf("claims JSON %s is missing %s", raw, field)
		}
	}
}
