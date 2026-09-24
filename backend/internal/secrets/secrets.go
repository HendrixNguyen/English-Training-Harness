// Package secrets seals the one secret the product stores on a user's behalf —
// users.google_refresh_token — with AES-256-GCM under ENCRYPTION_SECRET_KEY
// (backend spec §6.1, §7, §9). auth seals at sign-in, google opens at sync;
// neither package sees the key.
package secrets

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// KeyBytes is the AES-256 key length; ENCRYPTION_SECRET_KEY is its hex form (64 chars).
const KeyBytes = 32

// prefix marks a sealed value. A stored value without it is a pre-encryption
// plaintext row: Open refuses it (ErrNotSealed) rather than guessing.
const prefix = "v1:"

var (
	// ErrBadKey is a key that is not exactly KeyBytes long (or not valid hex).
	ErrBadKey = errors.New("secrets: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters")
	// ErrNotSealed is a value without the v1: prefix — a legacy plaintext row.
	ErrNotSealed = errors.New("secrets: value is not a v1 ciphertext")
	// ErrOpen is a v1 value that does not authenticate: tampered, truncated,
	// malformed, or sealed under another key.
	ErrOpen = errors.New("secrets: ciphertext did not authenticate")
)

// ParseHexKey decodes the 64-hex-character env form into 32 raw bytes.
func ParseHexKey(s string) ([]byte, error) {
	key, err := hex.DecodeString(strings.TrimSpace(s))
	if err != nil || len(key) != KeyBytes {
		return nil, ErrBadKey
	}
	return key, nil
}

// Box is an AES-256-GCM sealer/opener over one key. It satisfies auth.Sealer
// and google.Opener.
type Box struct{ aead cipher.AEAD }

// New builds a Box; key must be exactly KeyBytes long.
func New(key []byte) (*Box, error) {
	if len(key) != KeyBytes {
		return nil, ErrBadKey
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("secrets: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("secrets: %w", err)
	}
	return &Box{aead: aead}, nil
}

// Seal returns "v1:" + base64url(nonce ‖ ciphertext‖tag) with a fresh random
// 12-byte nonce, so sealing one plaintext twice yields two different values.
func (b *Box) Seal(plain string) (string, error) {
	nonce := make([]byte, b.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("secrets: nonce: %w", err)
	}
	sealed := b.aead.Seal(nonce, nonce, []byte(plain), nil) // nonce ‖ ct — Seal appends to its first argument
	return prefix + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// Open reverses Seal. ErrNotSealed for a value without the exact "v1:"
// prefix; ErrOpen for anything that does not decode and authenticate.
func (b *Box) Open(sealed string) (string, error) {
	if !strings.HasPrefix(sealed, prefix) {
		return "", ErrNotSealed
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(sealed, prefix))
	n := b.aead.NonceSize()
	if err != nil || len(raw) < n {
		return "", ErrOpen
	}
	plain, err := b.aead.Open(nil, raw[:n], raw[n:], nil)
	if err != nil {
		return "", ErrOpen
	}
	return string(plain), nil
}
