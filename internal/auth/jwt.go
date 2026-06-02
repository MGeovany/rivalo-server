// Package auth validates Supabase Auth JWTs and extracts the user identity.
package auth

import (
	"errors"
	"fmt"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
)

// ErrNotConfigured is returned when token verification is attempted without a
// configured key source.
var ErrNotConfigured = errors.New("auth: jwt verification not configured")

// Verifier validates Supabase Auth JWTs and extracts their subject claim.
//
// Supabase signs access tokens with asymmetric keys (ES256) by default, so the
// production verifier validates against the project's JSON Web Key Set. A
// symmetric (HS256) verifier is also provided for tests and legacy projects.
type Verifier struct {
	keyFunc jwt.Keyfunc
	methods []string
}

// NewVerifier builds an HS256 verifier from a shared secret. An empty secret
// yields an unconfigured verifier that rejects every token.
func NewVerifier(secret string) Verifier {
	if secret == "" {
		return Verifier{}
	}
	return Verifier{
		keyFunc: func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		},
		methods: []string{"HS256"},
	}
}

// NewJWKSVerifier builds a verifier that validates asymmetric tokens (ES256/
// RS256) against the JSON Web Key Set fetched from jwksURL. Keys are cached and
// refreshed in the background to handle rotation.
func NewJWKSVerifier(jwksURL string) (Verifier, error) {
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return Verifier{}, fmt.Errorf("load jwks: %w", err)
	}
	return Verifier{
		keyFunc: jwks.Keyfunc,
		methods: []string{"ES256", "RS256"},
	}, nil
}

// Configured reports whether a key source is present.
func (v Verifier) Configured() bool {
	return v.keyFunc != nil
}

// Verify checks the token's signature and expiry and returns its subject claim
// (the Supabase auth user id).
func (v Verifier) Verify(tokenString string) (string, error) {
	if v.keyFunc == nil {
		return "", ErrNotConfigured
	}

	token, err := jwt.Parse(tokenString, v.keyFunc, jwt.WithValidMethods(v.methods))
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
