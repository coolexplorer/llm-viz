package memory

import (
	"context"
	"sync"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
)

// KeyRepository is a thread-safe in-memory implementation of port.KeyRepository.
type KeyRepository struct {
	mu   sync.RWMutex
	keys map[string]domain.APIKey // keyID → APIKey
}

// NewKeyRepository creates an empty in-memory key repository.
func NewKeyRepository() *KeyRepository {
	return &KeyRepository{
		keys: make(map[string]domain.APIKey),
	}
}

// Save stores or replaces a key entry.
func (r *KeyRepository) Save(_ context.Context, key domain.APIKey) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.keys[key.ID] = key
	return nil
}

// Get retrieves a key by ID.
func (r *KeyRepository) Get(_ context.Context, id string) (*domain.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	k, ok := r.keys[id]
	if !ok {
		return nil, port.ErrKeyNotFound
	}
	return &k, nil
}

// GetByHash retrieves a key by its SHA-256 hash.
func (r *KeyRepository) GetByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, k := range r.keys {
		if k.KeyHash == hash {
			cp := k
			return &cp, nil
		}
	}
	return nil, port.ErrKeyNotFound
}

// List returns all keys, optionally filtered by provider.
// Passing an empty ProviderID returns keys for all providers.
func (r *KeyRepository) List(_ context.Context, provider domain.ProviderID) ([]domain.APIKey, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.APIKey, 0)
	for _, k := range r.keys {
		if provider == "" || k.Provider == provider {
			out = append(out, k)
		}
	}
	return out, nil
}

// Delete removes a key by ID.
func (r *KeyRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.keys[id]; !ok {
		return port.ErrKeyNotFound
	}
	delete(r.keys, id)
	return nil
}
