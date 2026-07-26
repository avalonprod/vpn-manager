package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nopLogger struct{}

func (nopLogger) Debug(...any)          {}
func (nopLogger) Debugf(string, ...any) {}
func (nopLogger) Info(...any)           {}
func (nopLogger) Infof(string, ...any)  {}
func (nopLogger) Warn(...any)           {}
func (nopLogger) Warnf(string, ...any)  {}
func (nopLogger) Error(...any)          {}
func (nopLogger) Errorf(string, ...any) {}

const webhookSecret = "cp-secret"

func signBody(body string) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(body))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func webhookRequest(t *testing.T, body, header, signature string) *httptest.ResponseRecorder {
	t.Helper()

	h := &Handler{cloudPaymentsSecret: webhookSecret, logger: nopLogger{}}

	reached := false
	guarded := h.authorizeCloudPayment(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		reached = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/cloudpayments/webhook/pay", strings.NewReader(body))
	if signature != "" {
		req.Header.Set(header, signature)
	}

	rec := httptest.NewRecorder()
	guarded.ServeHTTP(rec, req)

	if reached && rec.Code != http.StatusOK {
		t.Fatalf("handler ran but status is %d", rec.Code)
	}

	if !reached && rec.Code == http.StatusOK {
		t.Fatal("handler was skipped yet the response is 200")
	}

	return rec
}

func TestWebhookAcceptsValidSignature(t *testing.T) {
	body := "AccountId=1&InvoiceId=abc"

	if rec := webhookRequest(t, body, "Content-HMAC", signBody(body)); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200; a genuine CloudPayments callback was rejected", rec.Code)
	}
}

func TestWebhookAcceptsAlternateHeader(t *testing.T) {
	body := "AccountId=1&InvoiceId=abc"

	if rec := webhookRequest(t, body, "X-Content-HMAC", signBody(body)); rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestWebhookRejectsForgedCallback(t *testing.T) {
	cases := map[string]struct{ header, signature string }{
		"no signature":              {"Content-HMAC", ""},
		"wrong signature":           {"Content-HMAC", base64.StdEncoding.EncodeToString([]byte("not-a-real-mac-value-32bytes...."))},
		"not base64":                {"Content-HMAC", "%%%not-base64%%%"},
		"hex encoded":               {"Content-HMAC", "6162636465666768696a6b6c6d6e6f70"},
		"signature of another body": {"Content-HMAC", signBody("AccountId=999&InvoiceId=other")},
	}

	for name, c := range cases {
		rec := webhookRequest(t, "AccountId=1&InvoiceId=abc", c.header, c.signature)
		if rec.Code != http.StatusForbidden {
			t.Errorf("%s: status = %d, want 403 — a forged payment callback was accepted", name, rec.Code)
		}
	}
}

func TestWebhookLeavesBodyReadable(t *testing.T) {
	body := "AccountId=42&InvoiceId=xyz"

	h := &Handler{cloudPaymentsSecret: webhookSecret, logger: nopLogger{}}

	var seen string
	guarded := h.authorizeCloudPayment(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			return
		}
		seen = r.FormValue("AccountId")
	}))

	req := httptest.NewRequest(http.MethodPost, "/cloudpayments/webhook/pay", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Content-HMAC", signBody(body))

	guarded.ServeHTTP(httptest.NewRecorder(), req)

	if seen != "42" {
		t.Errorf("AccountId = %q, want 42; the guard consumed the body", seen)
	}
}
