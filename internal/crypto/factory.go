package crypto

import "fmt"

const (
	AlgorithmAESGCM           = "aes-gcm"
	AlgorithmChaCha20Poly1305 = "chacha20-poly1305"
)

// NewCipher builds the Cipher for the given algorithm name.
func NewCipher(algorithm, base64key string) (Cipher, error) {
	switch algorithm {
	case AlgorithmAESGCM:
		c, err := NewAESGCM(base64key)
		if err != nil {
			return nil, err
		}
		return c, nil
	case AlgorithmChaCha20Poly1305:
		c, err := NewChaChaPoly1305(base64key)
		if err != nil {
			return nil, err
		}
		return c, nil
	default:
		return nil, fmt.Errorf("unknown encryption algorithm %q", algorithm)
	}
}
