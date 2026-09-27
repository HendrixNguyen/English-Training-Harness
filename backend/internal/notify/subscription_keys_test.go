package notify

import (
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"
)

// validKeys generates a real P-256 key pair and a 16-byte auth secret the
// same way browserSubscription (push_test.go) does, returning them as the
// browser's unpadded base64url — what PushSubscription.toJSON().keys sends.
func validKeys(t *testing.T) (p256dh, auth string) {
	t.Helper()
	priv, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	a := make([]byte, 16)
	if _, err := rand.Read(a); err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(priv.PublicKey().Bytes()), base64.RawURLEncoding.EncodeToString(a)
}

func TestValidateSubscriptionKeysAcceptsBrowserKeys(t *testing.T) {
	p256dh, auth := validKeys(t)
	if err := ValidateSubscriptionKeys(p256dh, auth); err != nil {
		t.Errorf("unpadded base64url: err = %v", err)
	}

	// The same bytes, re-encoded with padded standard base64url.
	rawP, err := base64.RawURLEncoding.DecodeString(p256dh)
	if err != nil {
		t.Fatal(err)
	}
	rawA, err := base64.RawURLEncoding.DecodeString(auth)
	if err != nil {
		t.Fatal(err)
	}
	paddedP := base64.URLEncoding.EncodeToString(rawP)
	paddedA := base64.URLEncoding.EncodeToString(rawA)
	if err := ValidateSubscriptionKeys(paddedP, paddedA); err != nil {
		t.Errorf("padded base64url: err = %v", err)
	}
}

func TestValidateSubscriptionKeysRefusesBadMaterial(t *testing.T) {
	p256dh, auth := validKeys(t)
	rawP, _ := base64.RawURLEncoding.DecodeString(p256dh)
	rawA, _ := base64.RawURLEncoding.DecodeString(auth)

	off64 := make([]byte, 64)
	off66 := make([]byte, 66)
	compressed := append([]byte{0x02}, rawP[1:]...)
	offCurve := append([]byte{0x04}, make([]byte, 64)...) // all-zero point, not on P-256

	// 0xFF as the leading byte always encodes to '/' as the first standard-
	// base64 character (top 6 bits = 0b111111 = 63), so these are
	// deterministically "standard base64, not base64url" regardless of the
	// random bytes that follow — unlike re-encoding real key material, whose
	// standard-base64 form only sometimes differs from its base64url form.
	stdOnlyP := append([]byte{0xff}, rawP[1:]...)
	stdOnlyA := append([]byte{0xff}, rawA[1:]...)

	tests := map[string]struct {
		p256dh, auth string
	}{
		"p256dh 64 bytes":          {base64.RawURLEncoding.EncodeToString(off64), auth},
		"p256dh 66 bytes":          {base64.RawURLEncoding.EncodeToString(off66), auth},
		"p256dh compressed prefix": {base64.RawURLEncoding.EncodeToString(compressed), auth},
		"p256dh off curve":         {base64.RawURLEncoding.EncodeToString(offCurve), auth},
		"p256dh standard base64":   {base64.StdEncoding.EncodeToString(stdOnlyP), auth},
		"p256dh not base64":        {"not-valid-base64url!!", auth},
		"p256dh empty":             {"", auth},
		"auth 15 bytes":            {p256dh, base64.RawURLEncoding.EncodeToString(rawA[:15])},
		"auth 17 bytes":            {p256dh, base64.RawURLEncoding.EncodeToString(append(rawA, 0))},
		"auth standard base64":     {p256dh, base64.StdEncoding.EncodeToString(stdOnlyA)},
		"auth not base64":          {p256dh, "not-valid-base64url!!"},
		"auth empty":               {p256dh, ""},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateSubscriptionKeys(tt.p256dh, tt.auth)
			if !errors.Is(err, ErrInvalidSubscriptionKeys) {
				t.Errorf("err = %v, want ErrInvalidSubscriptionKeys", err)
			}
		})
	}
}

// A string containing standard-base64-only characters ('+' or '/') must be
// refused, not silently accepted by a lenient decoder.
func TestDecodeBase64URLRefusesStandardAlphabet(t *testing.T) {
	if _, err := decodeBase64URL("a+b/c"); err == nil {
		t.Error("standard-base64 characters must be refused")
	}
}
