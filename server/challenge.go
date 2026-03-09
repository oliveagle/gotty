package server

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"strings"
	"sync"
	"time"
)

// Challenge represents a pending authentication challenge
type Challenge struct {
	Value     string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// ChallengeManager manages authentication challenges
type ChallengeManager struct {
	challenges map[string]*Challenge
	mu         sync.RWMutex
	ttl        time.Duration
}

// NewChallengeManager creates a new challenge manager
func NewChallengeManager() *ChallengeManager {
	cm := &ChallengeManager{
		challenges: make(map[string]*Challenge),
		ttl:        5 * time.Minute, // Challenges expire after 5 minutes
	}
	// Start cleanup goroutine
	go cm.cleanupExpired()
	return cm
}

// Generate creates a new challenge for a session
func (cm *ChallengeManager) Generate(sessionID string) string {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Generate 32 random bytes
	bytes := make([]byte, 32)
	rand.Read(bytes)
	challengeValue := base64.StdEncoding.EncodeToString(bytes)

	now := time.Now()
	cm.challenges[sessionID] = &Challenge{
		Value:     challengeValue,
		CreatedAt: now,
		ExpiresAt: now.Add(cm.ttl),
	}

	return challengeValue
}

// Get retrieves a challenge for verification
func (cm *ChallengeManager) Get(sessionID string) (*Challenge, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	challenge, exists := cm.challenges[sessionID]
	if !exists {
		return nil, false
	}

	// Check if expired
	if time.Now().After(challenge.ExpiresAt) {
		return nil, false
	}

	return challenge, true
}

// Delete removes a challenge after use
func (cm *ChallengeManager) Delete(sessionID string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	delete(cm.challenges, sessionID)
}

// cleanupExpired periodically removes expired challenges
func (cm *ChallengeManager) cleanupExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		cm.mu.Lock()
		now := time.Now()
		for id, challenge := range cm.challenges {
			if now.After(challenge.ExpiresAt) {
				delete(cm.challenges, id)
			}
		}
		cm.mu.Unlock()
	}
}

// VerifySignature verifies an Ed25519 signature against a challenge
func VerifySignature(publicKeyBase64, challenge, signatureBase64 string) bool {
	// Decode public key
	publicKeyBytes, err := base64.StdEncoding.DecodeString(publicKeyBase64)
	if err != nil {
		return false
	}

	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return false
	}

	publicKey := ed25519.PublicKey(publicKeyBytes)

	// Decode signature
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false
	}

	// Verify signature
	return ed25519.Verify(publicKey, []byte(challenge), signature)
}
