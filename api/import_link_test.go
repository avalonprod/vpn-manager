package api

import (
	"net/url"
	"strings"
	"testing"
)

const testSubsURL = "https://api.neonguard.ru/subs?token=AbC-_123&name=NeonGuard"

func TestAndroidLinkKeepsSubscriptionOpaque(t *testing.T) {
	link := clientImportLink("android", testSubsURL)

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse %q: %v", link, err)
	}

	if parsed.Scheme != "hiddify" {
		t.Errorf("scheme = %q, want hiddify", parsed.Scheme)
	}

	got := parsed.Query().Get("url")
	if got != testSubsURL {
		t.Errorf("url = %q, want the subscription verbatim %q", got, testSubsURL)
	}

	if name := parsed.Query().Get("name"); name != profileName {
		t.Errorf("name = %q, want %q", name, profileName)
	}
}

func TestAndroidLinkHidesInnerQueryFromOuterURI(t *testing.T) {
	link := clientImportLink("android", testSubsURL)

	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if _, present := parsed.Query()["token"]; present {
		t.Error("token leaked into the outer link query — a re-normalising hop could drop it")
	}

	if strings.Count(link, "?") != 1 {
		t.Errorf("link %q has more than one raw '?', the inner URL is not escaped", link)
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

func TestUnsupportedOSYieldsNoLink(t *testing.T) {
	for _, os := range []string{"", "windows", "linux", "ANDROID"} {
		if link := clientImportLink(os, testSubsURL); link != "" {
			t.Errorf("os %q produced %q, want an empty link so the handler can refuse", os, link)
		}
	}
}
