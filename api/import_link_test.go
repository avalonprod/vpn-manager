package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testSubsURL = "https://api.neonguard.ru/subs/AbC-_123"

func TestAndroidLinkUsesHappAdd(t *testing.T) {
	link := clientImportLink("android", testSubsURL)

	if link != "happ://add/"+testSubsURL {
		t.Errorf("link = %q, want happ://add/ + the subscription", link)
	}
}

func TestAppleLinkKeepsExistingFormat(t *testing.T) {
	link := clientImportLink("ios", testSubsURL)

	if link != "streisand://import/"+testSubsURL {
		t.Errorf("link = %q, want the untouched streisand format", link)
	}

	if clientImportLink("macos", testSubsURL) != link {
		t.Error("macos must use the same link as ios")
	}
}

// v2rayN на Windows не регистрирует URL-схему, поэтому ссылки автоимпорта
// для него не существует — подписка добавляется вручную.
func TestWindowsHasNoImportLink(t *testing.T) {
	if link := clientImportLink("windows", testSubsURL); link != "" {
		t.Errorf("windows produced %q, want an empty link", link)
	}
}

func TestUnsupportedOSYieldsNoLink(t *testing.T) {
	for _, os := range []string{"", "linux", "ANDROID"} {
		if link := clientImportLink(os, testSubsURL); link != "" {
			t.Errorf("os %q produced %q, want an empty link so the handler can refuse", os, link)
		}
	}
}

func subscriptionURLFor(t *testing.T, target string) string {
	t.Helper()

	h := &Handler{apiUrl: "https://api.neonguard.ru"}

	return h.subscriptionURL(httptest.NewRequest(http.MethodGet, target, nil))
}

// Ссылка подписки не должна содержать query: у схем вида happ://add/<url>
// и streisand://import/<url> внутренний "?" становится query внешней ссылки
// и теряется на любом шаге, который нормализует URI.
func TestSubscriptionURLPutsTokenInThePath(t *testing.T) {
	got := subscriptionURLFor(t, "/setup?token=AbC-_123&os=android")

	if strings.Contains(got, "?") {
		t.Errorf("subscription url %q carries a query string", got)
	}

	if got != "https://api.neonguard.ru/subs/AbC-_123" {
		t.Errorf("subscription url = %q, want the token in the path", got)
	}
}

func TestSubscriptionURLEscapesToken(t *testing.T) {
	got := subscriptionURLFor(t, "/setup?token=a%2Fb%20c&os=android")

	if strings.Contains(got, "/subs/a/b") {
		t.Errorf("subscription url %q let the token add a path segment", got)
	}
}

func TestAndroidLinkHasNoQueryAtAll(t *testing.T) {
	link := clientImportLink("android", subscriptionURLFor(t, "/setup?token=AbC-_123&os=android"))

	if strings.Contains(link, "?") {
		t.Errorf("link %q contains a query string, nothing may split it", link)
	}

	if !strings.Contains(link, "AbC-_123") {
		t.Errorf("link %q lost the token", link)
	}
}
