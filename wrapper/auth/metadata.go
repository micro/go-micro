package auth

import (
	"errors"
	"strings"

	"go-micro.dev/v5/auth"
	"go-micro.dev/v5/metadata"
)

const (
	// MetadataKeyAuthorization is the key for the Authorization header in metadata
	MetadataKeyAuthorization = "Authorization"
	// BearerPrefix is the prefix for Bearer tokens
	BearerPrefix = "Bearer "
)

var (
	// ErrMissingToken is returned when no authorization token is found in metadata
	ErrMissingToken = errors.New("missing authorization token in metadata")
	// ErrInvalidToken is returned when the token format is invalid
	ErrInvalidToken = errors.New("invalid token format, expected 'Bearer <token>'")
)

// TokenFromMetadata extracts the Bearer token from request metadata.
// Returns the token string without the "Bearer " prefix, or an error if not found.
func TokenFromMetadata(md metadata.Metadata) (string, error) {
	// Check for Authorization header
	authHeader, ok := md.Get(MetadataKeyAuthorization)
	if !ok {
		// Also check lowercase version
		authHeader, ok = md.Get(strings.ToLower(MetadataKeyAuthorization))
		if !ok {
			return "", ErrMissingToken
		}
	}

	// Verify Bearer prefix
	if !strings.HasPrefix(authHeader, BearerPrefix) {
		return "", ErrInvalidToken
	}

	// Extract token (remove "Bearer " prefix)
	token := strings.TrimPrefix(authHeader, BearerPrefix)
	if token == "" {
		return "", ErrInvalidToken
	}

	return token, nil
}

// TokenToMetadata adds a Bearer token to metadata for outgoing requests.
// The token should be provided without the "Bearer " prefix.
func TokenToMetadata(md metadata.Metadata, token string) metadata.Metadata {
	if md == nil {
		md = metadata.Metadata{}
	}

	// Add Bearer prefix and set in metadata
	md.Set(MetadataKeyAuthorization, BearerPrefix+token)
	return md
}

// AccountFromMetadata extracts and verifies the token from metadata,
// returning the associated account. This is a convenience function that
// combines TokenFromMetadata and auth.Inspect.
func AccountFromMetadata(md metadata.Metadata, a auth.Auth) (*auth.Account, error) {
	token, err := TokenFromMetadata(md)
	if err != nil {
		return nil, err
	}

	return a.Inspect(token)
}
