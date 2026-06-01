// Package auth validates Supabase Auth JWTs and extracts the user identity.
package auth

import (
	"errors"
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// ErrNotConfigured is returned when token verification is attempted without a
// configured signing secret.
var ErrNotConfigured = errors.New("auth: jwt secret not configured")

// Verifier validates HS256-signed Supabase JWTs using the project's JWT secret.
type Verifier struct {
	secret []byte
}

// NewVerifier builds a Verifier from the Supabase JWT secret. An empty secret
// yields a verifier that rejects every token with ErrNotConfigured.
func NewVerifier(secret string) Verifier {
	return Verifier{secret: []byte(secret)}
}

// Configured reports whether a signing secret is present.
func (v Verifier) Configured() bool {
	return len(v.secret) > 0
}

// Verify checks the token's signature and expiry and returns its subject claim
// (the Supabase auth user id).
func (v Verifier) Verify(tokenString string) (string, error) {
	if !v.Configured() {
		return "", ErrNotConfigured
	}

	token, err := jwt.Parse(
		tokenString,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return v.secret, nil
		},
		jwt.WithValidMethods([]string{"HS256"}),
	)
	if err != nil {
		return "", err
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("auth: invalid token")
	}

	sub, _ := claims["sub"].(string)
	if sub == "" {
		return "", errors.New("auth: token has no subject")
	}
	return sub, nil
}
