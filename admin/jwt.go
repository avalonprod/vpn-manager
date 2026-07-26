package admin

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrTokenExpired = errors.New("token expired")
)

type Claims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`

	TokenID string `json:"jti"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func newTokenIssuer(secret string, ttl time.Duration) *tokenIssuer {
	return &tokenIssuer{secret: []byte(secret), ttl: ttl}
}

func encodeSegment(v any) (string, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func (t *tokenIssuer) sign(signingInput string) string {
	mac := hmac.New(sha256.New, t.secret)
	mac.Write([]byte(signingInput))

	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func newTokenID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {

		return ""
	}

	return hex.EncodeToString(buf)
}

func (t *tokenIssuer) Issue(subject string) (string, Claims, error) {
	now := time.Now().UTC()

	claims := Claims{
		Subject:   subject,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(t.ttl).Unix(),
		TokenID:   newTokenID(),
	}

	header, err := encodeSegment(jwtHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", Claims{}, err
	}

	payload, err := encodeSegment(claims)
	if err != nil {
		return "", Claims{}, err
	}

	signingInput := header + "." + payload

	return signingInput + "." + t.sign(signingInput), claims, nil
}

func (t *tokenIssuer) Parse(token string) (Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return Claims{}, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]

	if !hmac.Equal([]byte(parts[2]), []byte(t.sign(signingInput))) {
		return Claims{}, ErrInvalidToken
	}

	rawHeader, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var header jwtHeader
	if err := json.Unmarshal(rawHeader, &header); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if header.Algorithm != "HS256" {
		return Claims{}, ErrInvalidToken
	}

	rawPayload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(rawPayload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	if claims.ExpiresAt == 0 || time.Now().UTC().Unix() >= claims.ExpiresAt {
		return Claims{}, ErrTokenExpired
	}

	return claims, nil
}
