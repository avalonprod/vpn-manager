package secret

import (
	"encoding/base64"
	"strings"
	"testing"
)

const testKey = "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f"

func newTestCipher(t *testing.T) *Cipher {
	t.Helper()

	c, err := NewCipher(testKey)
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	return c
}

func TestRoundTrip(t *testing.T) {
	c := newTestCipher(t)

	for _, plaintext := range []string{"token", "3x-ui=abc.def", strings.Repeat("x", 500), "юникод"} {
		encrypted, err := c.Encrypt(plaintext)
		if err != nil {
			t.Fatalf("encrypt: %v", err)
		}

		if strings.Contains(encrypted, plaintext) {
			t.Errorf("ciphertext %q leaks the plaintext", encrypted)
		}

		decrypted, err := c.Decrypt(encrypted)
		if err != nil {
			t.Fatalf("decrypt: %v", err)
		}

		if decrypted != plaintext {
			t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
		}
	}
}

func TestEncryptUsesFreshNonce(t *testing.T) {
	c := newTestCipher(t)

	first, err := c.Encrypt("same-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	second, err := c.Encrypt("same-token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if first == second {
		t.Error("two encryptions of the same value produced identical ciphertext")
	}
}

func TestDecryptPassesThroughLegacyPlaintext(t *testing.T) {
	c := newTestCipher(t)

	got, err := c.Decrypt("plain-legacy-token")
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}

	if got != "plain-legacy-token" {
		t.Errorf("got %q, want the value unchanged", got)
	}
}

func TestDecryptRejectsTamperedCiphertext(t *testing.T) {
	c := newTestCipher(t)

	encrypted, err := c.Encrypt("token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(encrypted, prefix))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}

	raw[len(raw)-1] ^= 0xff

	tampered := prefix + base64.RawStdEncoding.EncodeToString(raw)

	if _, err := c.Decrypt(tampered); err != ErrCorruptCiphertext {
		t.Errorf("err = %v, want ErrCorruptCiphertext", err)
	}
}

func TestDecryptRejectsForeignKey(t *testing.T) {
	encrypted, err := newTestCipher(t).Encrypt("token")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	other, err := NewCipher(strings.Repeat("ff", KeyLen))
	if err != nil {
		t.Fatalf("new cipher: %v", err)
	}

	if _, err := other.Decrypt(encrypted); err != ErrCorruptCiphertext {
		t.Errorf("err = %v, want ErrCorruptCiphertext", err)
	}
}

func TestDecryptRejectsMalformedPayload(t *testing.T) {
	c := newTestCipher(t)

	for _, value := range []string{prefix + "!!!not-base64!!!", prefix + "", prefix + base64.RawStdEncoding.EncodeToString([]byte("short"))} {
		if _, err := c.Decrypt(value); err != ErrCorruptCiphertext {
			t.Errorf("Decrypt(%q) = %v, want ErrCorruptCiphertext", value, err)
		}
	}
}

func TestEmptyValueStaysEmpty(t *testing.T) {
	c := newTestCipher(t)

	encrypted, err := c.Encrypt("")
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}

	if encrypted != "" {
		t.Errorf("encrypted empty value = %q, want empty", encrypted)
	}
}

func TestNewCipherRejectsBadKeys(t *testing.T) {
	for _, key := range []string{"", "short", strings.Repeat("zz", KeyLen), strings.Repeat("ab", KeyLen-1)} {
		if _, err := NewCipher(key); err != ErrInvalidKey {
			t.Errorf("NewCipher(%q) = %v, want ErrInvalidKey", key, err)
		}
	}
}
