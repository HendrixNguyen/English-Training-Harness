package notify

import (
	"crypto/ecdh"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidSubscriptionKeys means p256dh or auth cannot possibly encrypt a
// push message: bad encoding, wrong length, or (for p256dh) not a point on
// P-256. UpdateSettings wraps it in ErrInvalidRequest (→ 400, nothing
// written) — storing key material that will fail at send time only earns
// the row a slow death through Tick's failure counter instead.
var ErrInvalidSubscriptionKeys = errors.New("notify: invalid push subscription keys")

// p256dhLen is an uncompressed P-256 point: 0x04 || X (32 bytes) || Y (32
// bytes), per SEC1 4.3.6.
const p256dhLen = 65

// authLen is the webpush auth secret (RFC 8291 §3.2).
const authLen = 16

// decodeBase64URL accepts both the browser's unpadded base64url
// (PushSubscription.toJSON() per RFC 7515 Appendix C) and the padded form,
// but never the standard alphabet ('+'/'/'): those characters are not valid
// base64url and must be refused, not silently reinterpreted.
func decodeBase64URL(s string) ([]byte, error) {
	if s == "" {
		return nil, errors.New("empty")
	}
	if strings.ContainsAny(s, "+/") {
		return nil, errors.New("standard base64 alphabet, want base64url")
	}
	trimmed := strings.TrimRight(s, "=")
	return base64.RawURLEncoding.DecodeString(trimmed)
}

// ValidateSubscriptionKeys checks that p256dh decodes to a 65-byte
// uncompressed point that actually lies on P-256, and auth decodes to
// exactly 16 bytes. Every failure wraps ErrInvalidSubscriptionKeys naming the
// field and why, but never echoes the value.
func ValidateSubscriptionKeys(p256dh, auth string) error {
	p, err := decodeBase64URL(p256dh)
	if err != nil {
		return fmt.Errorf("%w: p256dh: %v", ErrInvalidSubscriptionKeys, err)
	}
	if len(p) != p256dhLen {
		return fmt.Errorf("%w: p256dh: %d bytes, want %d", ErrInvalidSubscriptionKeys, len(p), p256dhLen)
	}
	if p[0] != 0x04 {
		return fmt.Errorf("%w: p256dh: not an uncompressed point (first byte 0x%02x)", ErrInvalidSubscriptionKeys, p[0])
	}
	if _, err := ecdh.P256().NewPublicKey(p); err != nil {
		return fmt.Errorf("%w: p256dh: not a valid P-256 point: %v", ErrInvalidSubscriptionKeys, err)
	}

	a, err := decodeBase64URL(auth)
	if err != nil {
		return fmt.Errorf("%w: auth: %v", ErrInvalidSubscriptionKeys, err)
	}
	if len(a) != authLen {
		return fmt.Errorf("%w: auth: %d bytes, want %d", ErrInvalidSubscriptionKeys, len(a), authLen)
	}
	return nil
}
