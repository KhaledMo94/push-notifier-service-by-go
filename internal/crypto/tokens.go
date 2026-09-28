package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const tokenPrefix = "pn_"

func GenerateToken() (plain, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}

	plain = tokenPrefix + base64.RawURLEncoding.EncodeToString(b)

	return plain, HashToken(plain), nil
}

func HashToken(plain string) (hashed string) {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
