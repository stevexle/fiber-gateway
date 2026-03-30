// Package utils provides common utilities for the Fiber Gateway.
// It includes cryptographic helpers, JWT management, and configuration loading.
package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
)

// CodeLength is the default bit-length for generating random PKCE strings.
const CodeLength = 32

// GenerateRandomString creates a URL-safe base64-encoded random string of the specified length.
// This is used for generating Secure Verifiers and Authorization Codes.
func GenerateRandomString(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// VerifyPKCE compares the code_verifier against the code_challenge using the specified method.
// Supported methods: "S256" (SHA256) and "plain".
// It returns an error if the verification fails or if the method is unsupported.
func VerifyPKCE(codeVerifier, codeChallenge, method string) error {
	if method == "S256" {
		hash := sha256.Sum256([]byte(codeVerifier))
		encoded := base64.RawURLEncoding.EncodeToString(hash[:])
		if encoded != codeChallenge {
			return errors.New("invalid code verifier")
		}
		return nil
	} else if method == "plain" || method == "" {
		if codeVerifier != codeChallenge {
			return errors.New("invalid code verifier")
		}
		return nil
	}
	return errors.New("unsupported code challenge method")
}
