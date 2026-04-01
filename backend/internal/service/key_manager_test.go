// Package service_test — API key management tests (Red phase, TDD).
//
// These tests define the expected behaviour of KeyManager before it is
// implemented. Running `go test ./internal/service/...` will produce compile
// errors until Task #2 creates service.KeyManager, domain.APIKey, and
// port.KeyRepository.
//
// Expected new types after implementation:
//
//	domain.APIKey        – stored key metadata (no raw key ever persisted)
//	port.KeyRepository   – CRUD interface for encrypted keys
//	service.KeyManager   – encryption/decryption, masking, hashing
package service_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kimseunghwan/llm-viz/backend/internal/domain"
	"github.com/kimseunghwan/llm-viz/backend/internal/port"
	"github.com/kimseunghwan/llm-viz/backend/internal/service"
)

// ---------------------------------------------------------------------------
// mockKeyRepo — implements port.KeyRepository for tests.
// This file intentionally references port.KeyRepository before it exists;
// compilation failure is the expected Red-phase signal.
// ---------------------------------------------------------------------------

type mockKeyRepo struct {
	keys    map[string]*domain.APIKey
	saveErr error
	getErr  error
	listErr error
	delErr  error
}

func newMockKeyRepo() *mockKeyRepo {
	return &mockKeyRepo{keys: make(map[string]*domain.APIKey)}
}

func (r *mockKeyRepo) Save(_ context.Context, key domain.APIKey) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.keys[key.ID] = &key
	return nil
}

func (r *mockKeyRepo) Get(_ context.Context, id string) (*domain.APIKey, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	k, ok := r.keys[id]
	if !ok {
		return nil, port.ErrKeyNotFound
	}
	return k, nil
}

func (r *mockKeyRepo) GetByHash(_ context.Context, hash string) (*domain.APIKey, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	for _, k := range r.keys {
		if k.KeyHash == hash {
			return k, nil
		}
	}
	return nil, port.ErrKeyNotFound
}

func (r *mockKeyRepo) List(_ context.Context, provider domain.ProviderID) ([]domain.APIKey, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []domain.APIKey
	for _, k := range r.keys {
		if provider == "" || k.Provider == provider {
			out = append(out, *k)
		}
	}
	return out, nil
}

func (r *mockKeyRepo) Delete(_ context.Context, id string) error {
	if r.delErr != nil {
		return r.delErr
	}
	if _, ok := r.keys[id]; !ok {
		return port.ErrKeyNotFound
	}
	delete(r.keys, id)
	return nil
}

// validEncryptionKey is a 32-byte key required for AES-256-GCM.
var validEncryptionKey = []byte("secret-key-32-bytes-long-here!!")

// ---------------------------------------------------------------------------
// NewKeyManager — construction
// ---------------------------------------------------------------------------

func TestNewKeyManager_Success(t *testing.T) {
	repo := newMockKeyRepo()
	km, err := service.NewKeyManager(repo, validEncryptionKey)
	require.NoError(t, err)
	assert.NotNil(t, km)
}

func TestNewKeyManager_NilRepo(t *testing.T) {
	_, err := service.NewKeyManager(nil, validEncryptionKey)
	assert.Error(t, err)
}

func TestNewKeyManager_ShortKey(t *testing.T) {
	// AES-256 requires exactly 32 bytes.
	_, err := service.NewKeyManager(newMockKeyRepo(), []byte("tooshort"))
	assert.Error(t, err)
}

func TestNewKeyManager_EmptyEncryptionKey(t *testing.T) {
	_, err := service.NewKeyManager(newMockKeyRepo(), []byte{})
	assert.Error(t, err)
}

func TestNewKeyManager_16ByteKey(t *testing.T) {
	// AES-128 key (16 bytes) should be rejected for AES-256.
	_, err := service.NewKeyManager(newMockKeyRepo(), []byte("exactly-16-bytes"))
	assert.Error(t, err, "must require exactly 32-byte key for AES-256")
}

// ---------------------------------------------------------------------------
// SaveAPIKey — happy path
// ---------------------------------------------------------------------------

func TestKeyManager_SaveAPIKey_Success(t *testing.T) {
	km, repo := newKeyManagerWithRepo(t)
	ctx := context.Background()

	key, err := km.SaveAPIKey(ctx, domain.ProviderAnthropic, "My Anthropic Key", "sk-ant-api03-testkey123")
	require.NoError(t, err)
	require.NotNil(t, key)

	assert.NotEmpty(t, key.ID)
	assert.Equal(t, domain.ProviderAnthropic, key.Provider)
	assert.Equal(t, "My Anthropic Key", key.Name)
	assert.NotEmpty(t, key.MaskedKey)
	assert.NotEmpty(t, key.KeyHash)
	assert.False(t, key.CreatedAt.IsZero())

	// Raw key must never be stored.
	assert.Empty(t, key.RawKey, "RawKey field must not be populated on returned key")
	// Repo should hold encrypted blob, not the plaintext.
	saved := repo.keys[key.ID]
	require.NotNil(t, saved)
	assert.NotEmpty(t, saved.EncryptedKey)
}

func TestKeyManager_SaveAPIKey_PersistsToRepo(t *testing.T) {
	km, repo := newKeyManagerWithRepo(t)
	key, err := km.SaveAPIKey(context.Background(), domain.ProviderOpenAI, "GPT Key", "sk-proj-abc123")
	require.NoError(t, err)
	assert.Len(t, repo.keys, 1)
	assert.Equal(t, key.ID, repo.keys[key.ID].ID)
}

func TestKeyManager_SaveAPIKey_CreatesUniqueIDs(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()

	k1, err := km.SaveAPIKey(ctx, domain.ProviderOpenAI, "Key 1", "sk-proj-aaa111")
	require.NoError(t, err)
	k2, err := km.SaveAPIKey(ctx, domain.ProviderOpenAI, "Key 2", "sk-proj-bbb222")
	require.NoError(t, err)

	assert.NotEqual(t, k1.ID, k2.ID)
}

// ---------------------------------------------------------------------------
// SaveAPIKey — validation errors
// ---------------------------------------------------------------------------

func TestKeyManager_SaveAPIKey_EmptyKey(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	_, err := km.SaveAPIKey(context.Background(), domain.ProviderAnthropic, "name", "")
	assert.Error(t, err)
}

func TestKeyManager_SaveAPIKey_EmptyName(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	_, err := km.SaveAPIKey(context.Background(), domain.ProviderAnthropic, "", "sk-ant-api03-key")
	assert.Error(t, err)
}

func TestKeyManager_SaveAPIKey_EmptyProvider(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	_, err := km.SaveAPIKey(context.Background(), "", "name", "sk-key-abc")
	assert.Error(t, err)
}

func TestKeyManager_SaveAPIKey_InvalidProvider(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	_, err := km.SaveAPIKey(context.Background(), "nonexistent", "name", "sk-key-abc")
	assert.Error(t, err)
}

func TestKeyManager_SaveAPIKey_RepoError(t *testing.T) {
	repo := newMockKeyRepo()
	repo.saveErr = assert.AnError
	km, err := service.NewKeyManager(repo, validEncryptionKey)
	require.NoError(t, err)

	_, err = km.SaveAPIKey(context.Background(), domain.ProviderOpenAI, "name", "sk-proj-abc")
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// GetDecryptedKey
// ---------------------------------------------------------------------------

func TestKeyManager_GetDecryptedKey_Success(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()
	rawKey := "sk-ant-api03-supersecret"

	saved, err := km.SaveAPIKey(ctx, domain.ProviderAnthropic, "test", rawKey)
	require.NoError(t, err)

	decrypted, err := km.GetDecryptedKey(ctx, saved.ID)
	require.NoError(t, err)
	assert.Equal(t, rawKey, decrypted, "decrypted key must match original")
}

func TestKeyManager_GetDecryptedKey_NotFound(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	_, err := km.GetDecryptedKey(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, port.ErrKeyNotFound)
}

func TestKeyManager_GetDecryptedKey_EncryptionRoundTrip(t *testing.T) {
	// Verify that AES-256-GCM encrypt → persist → decrypt is lossless.
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()
	originals := []string{
		"sk-ant-api03-short",
		"sk-proj-" + strings.Repeat("x", 64),
		"gsk_" + strings.Repeat("a", 52),
	}
	for _, raw := range originals {
		saved, err := km.SaveAPIKey(ctx, domain.ProviderAnthropic, "k", raw)
		require.NoError(t, err)
		got, err := km.GetDecryptedKey(ctx, saved.ID)
		require.NoError(t, err)
		assert.Equal(t, raw, got)
	}
}

// ---------------------------------------------------------------------------
// ListKeys
// ---------------------------------------------------------------------------

func TestKeyManager_ListKeys_EmptyResult(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	keys, err := km.ListKeys(context.Background(), domain.ProviderOpenAI)
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestKeyManager_ListKeys_FiltersByProvider(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()

	_, _ = km.SaveAPIKey(ctx, domain.ProviderOpenAI, "oai", "sk-proj-111")
	_, _ = km.SaveAPIKey(ctx, domain.ProviderAnthropic, "ant", "sk-ant-api03-222")

	oaiKeys, err := km.ListKeys(ctx, domain.ProviderOpenAI)
	require.NoError(t, err)
	assert.Len(t, oaiKeys, 1)
	assert.Equal(t, domain.ProviderOpenAI, oaiKeys[0].Provider)
}

func TestKeyManager_ListKeys_AllProviders(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()

	_, _ = km.SaveAPIKey(ctx, domain.ProviderOpenAI, "oai", "sk-proj-111")
	_, _ = km.SaveAPIKey(ctx, domain.ProviderAnthropic, "ant", "sk-ant-api03-222")

	all, err := km.ListKeys(ctx, "")
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestKeyManager_ListKeys_NeverReturnsRawKey(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()
	_, _ = km.SaveAPIKey(ctx, domain.ProviderOpenAI, "k", "sk-proj-secret-value")

	keys, err := km.ListKeys(ctx, domain.ProviderOpenAI)
	require.NoError(t, err)
	require.Len(t, keys, 1)
	assert.Empty(t, keys[0].RawKey, "RawKey must never be returned from ListKeys")
}

// ---------------------------------------------------------------------------
// DeleteKey
// ---------------------------------------------------------------------------

func TestKeyManager_DeleteKey_Success(t *testing.T) {
	km, repo := newKeyManagerWithRepo(t)
	ctx := context.Background()

	saved, err := km.SaveAPIKey(ctx, domain.ProviderOpenAI, "name", "sk-proj-deleteme")
	require.NoError(t, err)

	err = km.DeleteKey(ctx, saved.ID)
	require.NoError(t, err)
	assert.Empty(t, repo.keys, "repo must be empty after delete")
}

func TestKeyManager_DeleteKey_NotFound(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	err := km.DeleteKey(context.Background(), "ghost-id")
	assert.ErrorIs(t, err, port.ErrKeyNotFound)
}

// ---------------------------------------------------------------------------
// MaskKey
// ---------------------------------------------------------------------------

func TestKeyManager_MaskKey_AnthropicFormat(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	// "sk-ant-api03-xxx...yyy" → "sk-ant-***yyy"
	masked := km.MaskKey("sk-ant-api03-abcdefghij")
	assert.Contains(t, masked, "***")
	assert.False(t, strings.Contains(masked, "abcdefg"), "middle portion must be redacted")
	assert.True(t, strings.HasSuffix(masked, masked[len(masked)-4:]))
}

func TestKeyManager_MaskKey_OpenAIFormat(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	// "sk-proj-xxx...yyy" → shows prefix + *** + last4
	masked := km.MaskKey("sk-proj-abcdefghij1234")
	assert.Contains(t, masked, "***")
	assert.True(t, strings.HasSuffix(masked, "1234"))
}

func TestKeyManager_MaskKey_ShortKey(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	// Keys shorter than 8 chars must be fully masked.
	masked := km.MaskKey("short")
	assert.Equal(t, "***", masked)
}

func TestKeyManager_MaskKey_EmptyKey(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	masked := km.MaskKey("")
	assert.Equal(t, "***", masked)
}

// ---------------------------------------------------------------------------
// Hash generation
// ---------------------------------------------------------------------------

func TestKeyManager_HashIsConsistent(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	ctx := context.Background()

	raw := "sk-proj-hashme-consistent"
	k1, err := km.SaveAPIKey(ctx, domain.ProviderOpenAI, "a", raw)
	require.NoError(t, err)
	k2, err := km.SaveAPIKey(ctx, domain.ProviderOpenAI, "b", raw)
	require.NoError(t, err)

	assert.Equal(t, k1.KeyHash, k2.KeyHash, "same raw key must always produce same hash")
}

func TestKeyManager_HashIsNotRawKey(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	raw := "sk-proj-supersecret-value"
	k, err := km.SaveAPIKey(context.Background(), domain.ProviderOpenAI, "name", raw)
	require.NoError(t, err)
	assert.NotEqual(t, raw, k.KeyHash, "hash must not equal plaintext key")
}

// ---------------------------------------------------------------------------
// Timestamps
// ---------------------------------------------------------------------------

func TestKeyManager_SaveAPIKey_SetsCreatedAt(t *testing.T) {
	km, _ := newKeyManagerWithRepo(t)
	before := time.Now()
	k, err := km.SaveAPIKey(context.Background(), domain.ProviderOpenAI, "ts", "sk-proj-time")
	require.NoError(t, err)
	after := time.Now()
	assert.True(t, k.CreatedAt.After(before) || k.CreatedAt.Equal(before))
	assert.True(t, k.CreatedAt.Before(after) || k.CreatedAt.Equal(after))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newKeyManagerWithRepo(t *testing.T) (*service.KeyManager, *mockKeyRepo) {
	t.Helper()
	repo := newMockKeyRepo()
	km, err := service.NewKeyManager(repo, validEncryptionKey)
	require.NoError(t, err)
	return km, repo
}
