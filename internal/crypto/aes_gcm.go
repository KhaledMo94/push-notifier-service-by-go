package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
)

const aesGCMVersion = "aesgcmv1:"

type AESGCM struct {
	aead cipher.AEAD
}

var _ Cipher = (*AESGCM)(nil)

func NewAESGCM(base64key string) (*AESGCM, error) {
	key, err := base64.StdEncoding.DecodeString(base64key)
	if err != nil {
		return nil, fmt.Errorf("decode key: %w", err)
	}

	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes, got %d", len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &AESGCM{aead: aead}, nil
}

func (c *AESGCM) Encrypt(plaintext []byte) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	sealed := c.aead.Seal(nonce, nonce, plaintext, nil)

	return aesGCMVersion + base64.StdEncoding.EncodeToString(sealed), nil
}

func (c *AESGCM) Decrypt(encoded string) ([]byte, error) {
	raw, ok := strings.CutPrefix(encoded, aesGCMVersion)
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
