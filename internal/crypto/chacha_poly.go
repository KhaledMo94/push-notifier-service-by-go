package crypto

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"

	"golang.org/x/crypto/chacha20poly1305"
)

const chachaVersion = "chachav1:"

type Chacha20poly1305 struct {
	aead cipher.AEAD
}

var _ Cipher = (*Chacha20poly1305)(nil)

func NewChaChaPoly1305(base64key string) (*Chacha20poly1305, error) {
	key, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	if len(key) != chacha20poly1305.KeySize {
		return nil, fmt.Errorf("key must be %d bytes, got %d", chacha20poly1305.KeySize, len(key))
	}

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, err
	}

	return &Chacha20poly1305{aead: aead}, nil
}

func (c *Chacha20poly1305) GenerateNonce() ([]byte, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return nonce, nil
}

func (c *Chacha20poly1305) Encrypt(plaintext []byte) (string, error) {
	nonce, err := c.GenerateNonce()
	if err != nil {
		return "", err
	}

	sealed := c.aead.Seal(nonce, nonce, plaintext, nil)

	return chachaVersion + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *Chacha20poly1305) Decrypt(encoded string) ([]byte, error) {
	raw, ok := strings.CutPrefix(encoded, chachaVersion)
	if !ok {
		return nil, ErrInvalidCiphertext
	}

	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil || len(data) < c.aead.NonceSize() {
		return nil, ErrInvalidCiphertext
	}

	nonce, ciphertext := data[:c.aead.NonceSize()], data[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	return plaintext, nil
}
