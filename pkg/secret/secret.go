package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

const (
	KeyLen  = 32
	prefix  = "v1:"
	nonceOK = 12
)

var (
	ErrInvalidKey        = errors.New("encryption key must be 32 bytes hex-encoded (64 characters)")
	ErrCorruptCiphertext = errors.New("ciphertext is corrupt or encrypted with a different key")
)

type Cipher struct {
	aead cipher.AEAD
}

func NewCipher(hexKey string) (*Cipher, error) {
	key, err := hex.DecodeString(strings.TrimSpace(hexKey))
	if err != nil || len(key) != KeyLen {
		return nil, ErrInvalidKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("init aes: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("init gcm: %w", err)
	}

	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)

	return prefix + base64.RawStdEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(value string) (string, error) {
	encoded, encrypted := strings.CutPrefix(value, prefix)
	if !encrypted {
		return value, nil
	}

	sealed, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrCorruptCiphertext
	}

	if len(sealed) < c.aead.NonceSize() {
		return "", ErrCorruptCiphertext
	}

	nonce, ciphertext := sealed[:c.aead.NonceSize()], sealed[c.aead.NonceSize():]

	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", ErrCorruptCiphertext
	}

	return string(plaintext), nil
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, prefix)
}
