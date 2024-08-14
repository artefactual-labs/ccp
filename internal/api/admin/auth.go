package admin

import (
	"context"
	"strings"

	"connectrpc.com/authn"
	"github.com/go-logr/logr"

	"github.com/artefactual-labs/ccp/internal/store"
)

var errInvalidAuth = authn.Errorf("invalid authorization")

func authenticate(logger logr.Logger, store store.Store) authn.AuthFunc {
	return multiAuthenticate(
		authApiKey(logger, store),
	)
}

func multiAuthenticate(methods ...authn.AuthFunc) authn.AuthFunc {
	return func(ctx context.Context, req authn.Request) (any, error) {
		var lastErr error
		for _, method := range methods {
			result, err := method(ctx, req)
			if err == nil {
				return result, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

func authApiKey(logger logr.Logger, store store.Store) authn.AuthFunc {
	return func(ctx context.Context, req authn.Request) (any, error) {
		auth := req.Header().Get("Authorization")
		if auth == "" {
			return nil, errInvalidAuth
		}

		username, key, ok := parseApiKey(auth)
		if !ok {
			return nil, errInvalidAuth
		}

		user, err := store.ValidateUserAPIKey(ctx, username, key)
		if err != nil {
			logger.Error(err, "Cannot look up user details.")
			return nil, errInvalidAuth
		}
		if user == nil {
			return nil, errInvalidAuth
		}

		return user, nil
	}
}

// parseApiKey parses the ApiKey string.
// "ApiKey test:test" returns ("test", "test", true).
func parseApiKey(auth string) (username, key string, ok bool) {
	const prefix = "ApiKey "
	// Case insensitive prefix match.
	if len(auth) < len(prefix) || !equalFold(auth[:len(prefix)], prefix) {
		return "", "", false
	}
	username, key, ok = strings.Cut(auth[len(prefix):], ":")
	if !ok {
		return "", "", false
	}
	return username, key, true
}

// equalFold is [strings.EqualFold], ASCII only. It reports whether s and t
// are equal, ASCII-case-insensitively.
func equalFold(s, t string) bool {
	if len(s) != len(t) {
		return false
	}
	for i := range len(s) {
		if lower(s[i]) != lower(t[i]) {
			return false
		}
	}
	return true
}

// lower returns the ASCII lowercase version of b.
func lower(b byte) byte {
	if 'A' <= b && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}
