package server

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"io"
)

// BitwardenAuthConfig holds configuration for bitwarden authentication
type BitwardenAuthConfig struct {
	// SecretKey is used to verify the derived key from client
	// In production, this could be stored in a vault or config
	SecretKey string
}

// DeriveKeyFromPassword derives a key from password and email using PBKDF2
// This matches the client's key derivation
// WARNING: This is a simplified implementation. For production, use proper PBKDF2.
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
// Uses constant-time comparison to prevent timing attacks
func (config *BitwardenAuthConfig) VerifyToken(token string) bool {
	if config.SecretKey == "" {
		// No secret configured, reject all tokens for security
		return false
	}
	// Security: Use constant-time comparison
	return subtle.ConstantTimeCompare([]byte(config.SecretKey), []byte(token)) == 1
}

// EncryptData encrypts data using AES-256-GCM
func EncryptData(plaintext, key string) (string, error) {
	keyBytes := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}

	// Use GCM mode for authenticated encryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Generate random nonce (IV)
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// Encrypt and seal
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptData decrypts data using AES-256-GCM
func DecryptData(ciphertext, key string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("empty ciphertext")
	}

	keyBytes := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return "", err
	}

	// Use GCM mode for authenticated decryption
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// Decode base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	// Extract nonce
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt and verify
	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
