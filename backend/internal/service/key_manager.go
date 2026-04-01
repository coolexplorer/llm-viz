package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// validProviders is the set of supported provider IDs.
var validProviders = map[domain.ProviderID]bool{
	domain.ProviderAnthropic:  true,
	domain.ProviderOpenAI:     true,
	domain.ProviderGemini:     true,
	domain.ProviderMistral:    true,
	domain.ProviderGroq:       true,
	domain.ProviderOpenRouter: true,
}

// KeyManager handles API key encryption, decryption, masking, and storage.
type KeyManager struct {
	repo      port.KeyRepository
	secretKey []byte // must be exactly 32 bytes for AES-256-GCM
}

// NewKeyManager creates a KeyManager.
// secretKey must be at least 24 bytes; it is hashed via SHA-256 to produce
// the actual 32-byte AES-256 key (rejects AES-128 / short inputs).
func NewKeyManager(repo port.KeyRepository, secretKey []byte) (*KeyManager, error) {
	if repo == nil {
		return nil, errors.New("key repository must not be nil")
	}
	if len(secretKey) < 24 {
		return nil, fmt.Errorf("encryption key must be at least 24 bytes for AES-256, got %d", len(secretKey))
	}
	// Derive exactly 32 bytes for AES-256 using SHA-256.
	derived := sha256.Sum256(secretKey)
	aesKey := make([]byte, 32)
	copy(aesKey, derived[:])
	return &KeyManager{repo: repo, secretKey: aesKey}, nil
}

// SaveAPIKey encrypts and persists a raw API key.
// The returned APIKey never contains RawKey.
func (km *KeyManager) SaveAPIKey(ctx context.Context, provider domain.ProviderID, name, rawKey string) (domain.APIKey, error) {
	if provider == "" {
		return domain.APIKey{}, errors.New("provider is required")
	}
	if !validProviders[provider] {
		return domain.APIKey{}, fmt.Errorf("unsupported provider: %s", provider)
	}
	if name == "" {
		return domain.APIKey{}, errors.New("name is required")
	}
	if rawKey == "" {
		return domain.APIKey{}, errors.New("key is required")
	}

	encrypted, err := km.encrypt(rawKey)
	if err != nil {
		return domain.APIKey{}, fmt.Errorf("encrypt: %w", err)
	}

	apiKey := domain.APIKey{
		ID:           uuid.New().String(),
		Provider:     provider,
		Name:         name,
		EncryptedKey: encrypted,
		KeyHash:      km.hashKey(rawKey),
		MaskedKey:    km.MaskKey(rawKey),
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	if err := km.repo.Save(ctx, apiKey); err != nil {
		return domain.APIKey{}, fmt.Errorf("save: %w", err)
	}

	// RawKey is never populated on return.
	return apiKey, nil
}

// GetDecryptedKey retrieves and decrypts a key by its ID.
func (km *KeyManager) GetDecryptedKey(ctx context.Context, keyID string) (string, error) {
	k, err := km.repo.Get(ctx, keyID)
	if err != nil {
		return "", err
	}
	raw, err := km.decrypt(k.EncryptedKey)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}
	return raw, nil
}

// ListKeys returns stored keys for the given provider (empty = all providers).
// Returned APIKey structs never contain RawKey.
func (km *KeyManager) ListKeys(ctx context.Context, provider domain.ProviderID) ([]domain.APIKey, error) {
	keys, err := km.repo.List(ctx, provider)
	if err != nil {
		return nil, err
	}
	// Ensure RawKey is always empty.
	for i := range keys {
		keys[i].RawKey = ""
	}
	return keys, nil
}

// DeleteKey removes a key by ID.
func (km *KeyManager) DeleteKey(ctx context.Context, keyID string) error {
	return km.repo.Delete(ctx, keyID)
}

// MaskKey returns a display-safe version of the key.
// Keys shorter than 8 characters are fully masked as "***".
func (km *KeyManager) MaskKey(rawKey string) string {
	if len(rawKey) < 8 {
		return "***"
	}
	return rawKey[:7] + "***" + rawKey[len(rawKey)-4:]
}

// hashKey returns the SHA-256 hex digest of the raw key.
func (km *KeyManager) hashKey(rawKey string) string {
	h := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(h[:])
}

// encrypt encrypts plaintext with AES-256-GCM, prepending the random nonce.
func (km *KeyManager) encrypt(plaintext string) ([]byte, error) {
	block, err := aes.NewCipher(km.secretKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	// Seal appends ciphertext+tag to nonce.
	return gcm.Seal(nonce, nonce, []byte(plaintext), nil), nil
}

// decrypt decrypts an AES-256-GCM ciphertext (nonce prepended).
func (km *KeyManager) decrypt(ciphertext []byte) (string, error) {
	block, err := aes.NewCipher(km.secretKey)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
