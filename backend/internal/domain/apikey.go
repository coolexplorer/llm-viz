package domain

import "time"

// APIKey holds encrypted API key metadata.
// The raw key is never persisted; only EncryptedKey is stored.
type APIKey struct {
	ID           string
	Provider     ProviderID
	Name         string
	EncryptedKey []byte     // AES-256-GCM encrypted key blob
	KeyHash      string     // SHA-256 hex hash of the raw key (for dedup lookup)
	MaskedKey    string     // e.g. "sk-ant-***xyzw"
	RawKey       string     // NEVER persisted; always empty on returned structs
	IsActive     bool
	CreatedAt    time.Time
}
