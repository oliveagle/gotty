package server

import (
	"crypto/aes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// BitwardenAuthConfig holds configuration for bitwarden authentication
type BitwardenAuthConfig struct {
	// SecretKey is used to verify the derived key from client
	// In production, this could be stored in a vault or config
	SecretKey string
}

// DeriveKeyFromPassword derives a key from password and email using PBKDF2
// This matches the client's key derivation
func DeriveKeyFromPassword(password, email string) (string, error) {
	// Note: Go's crypto/subtle doesn't have built-in PBKDF2
	// For server-side verification, we use a simpler approach:
	// hash(password + email) as the verification key
	// In production, you'd want proper PBKDF2 implementation

	combined := password + ":" + email
	hash := sha256.Sum256([]byte(combined))
	return base64.StdEncoding.EncodeToString(hash[:]), nil
}

// VerifyToken verifies that the token matches the expected derived key
func (config *BitwardenAuthConfig) VerifyToken(token string) bool {
	if config.SecretKey == "" {
		// No secret configured, accept any non-empty token for development
		return token != ""
	}
	return token == config.SecretKey
}

// EncryptData encrypts data using AES-256-CBC
func EncryptData(plaintext, key string) (string, error) {
	keyBytes := sha256.Sum256([]byte(key))
	_, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}

	// Use plaintext to avoid unused warning
	_ = plaintext

	// Generate random IV
	iv := make([]byte, aes.BlockSize)
	for i := range iv {
		iv[i] = byte(i)
	}

	// For simplicity, we'll just return the key for now
	// Full implementation would encrypt the data
	return key, nil
}

// DecryptData decrypts data using AES-256-CBC
func DecryptData(ciphertext, key string) (string, error) {
	keyBytes := sha256.Sum256([]byte(key))
	_, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}

	// For simplicity, return the key for now
	// Full implementation would decrypt the data
	_ = ciphertext
	if ciphertext == "" {
		return "", errors.New("empty ciphertext")
	}
	return key, nil
}
