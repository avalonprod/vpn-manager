package servers

import (
	"strings"
	"testing"
	"vpn-manager/pkg/secret"

	"go.mongodb.org/mongo-driver/bson"
)

const storeTestKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func newTestStore(t *testing.T) *store {
	t.Helper()

	cipher, err := secret.NewCipher(storeTestKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	return &store{cipher: cipher}
}

func TestCreateEncryptsTokenBeforeInsert(t *testing.T) {
	s := newTestStore(t)

	server := Server{AuthToken: "panel-token"}

	stored, err := s.cipher.Encrypt(server.AuthToken)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if !secret.IsEncrypted(stored) {
		t.Fatalf("stored value %q is not marked as encrypted", stored)
	}

	if strings.Contains(stored, "panel-token") {
		t.Errorf("stored value %q leaks the token", stored)
	}
}

func TestUpdateEncryptsTokenField(t *testing.T) {
	s := newTestStore(t)

	fields := bson.M{"auth_token": "new-token", "location": "Berlin"}

	if raw, ok := fields["auth_token"].(string); ok {
		encrypted, err := s.cipher.Encrypt(raw)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}
		fields["auth_token"] = encrypted
	}

	token, ok := fields["auth_token"].(string)
	if !ok || !secret.IsEncrypted(token) {
		t.Fatalf("auth_token = %v, want an encrypted value", fields["auth_token"])
	}

	if fields["location"] != "Berlin" {
		t.Errorf("location = %v, want it untouched", fields["location"])
	}
}

func TestDecodeTokenRestoresPlaintext(t *testing.T) {
	s := newTestStore(t)

	encrypted, err := s.cipher.Encrypt("panel-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	server := Server{ID: "srv1", AuthToken: encrypted}

	if err := s.decodeToken(&server); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if server.AuthToken != "panel-token" {
		t.Errorf("AuthToken = %q, want %q", server.AuthToken, "panel-token")
	}
}

func TestDecodeTokenAcceptsLegacyPlaintext(t *testing.T) {
	s := newTestStore(t)

	server := Server{ID: "srv1", AuthToken: "legacy-plain-token"}

	if err := s.decodeToken(&server); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if server.AuthToken != "legacy-plain-token" {
		t.Errorf("AuthToken = %q, want the legacy value unchanged", server.AuthToken)
	}
}

func TestDecodeTokenFailsOnWrongKey(t *testing.T) {
	encrypted, err := newTestStore(t).cipher.Encrypt("panel-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	otherCipher, err := secret.NewCipher(strings.Repeat("ab", secret.KeyLen))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	other := &store{cipher: otherCipher}
	server := Server{ID: "srv1", AuthToken: encrypted}

	err = other.decodeToken(&server)
	if err == nil {
		t.Fatal("decoding with the wrong key silently succeeded")
	}

	if !strings.Contains(err.Error(), "srv1") {
		t.Errorf("error %q does not name the offending server", err)
	}
}
