package domain

import "errors"

var (
	ErrUnknownProvider     = errors.New("unknown provider")
	ErrProviderUnavailable = errors.New("provider unavailable")
	ErrRateLimited         = errors.New("rate limited by provider")
	ErrInvalidAPIKey       = errors.New("invalid API key")
	ErrContextExceeded     = errors.New("context window exceeded")
	ErrModelNotFound       = errors.New("model not found")
)
