package crypto

import "errors"

type Cipher interface {
	Encrypt(plaintext []byte) (string, error)
	Decrypt(encoded string) ([]byte, error)
	GenerateNonce() ([]byte, error)
}

var ErrInvalidCiphertext = errors.New("invalid ciphertext")
