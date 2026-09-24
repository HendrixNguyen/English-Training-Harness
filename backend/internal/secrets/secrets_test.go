package secrets

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func testKey(t *testing.T) []byte {
	t.Helper()
	key := make([]byte, KeyBytes)
	for i := range key {
		key[i] = byte(i)
	}
	return key
}

func newBox(t *testing.T, key []byte) *Box {
	t.Helper()
	b, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return b
}

func TestSealThenOpenRoundTrips(t *testing.T) {
	b := newBox(t, testKey(t))
	sealed, err := b.Seal("1//refresh-token")
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if !strings.HasPrefix(sealed, "v1:") || strings.Contains(sealed, "refresh") {
		t.Errorf("sealed = %q, want a v1: ciphertext that does not contain the plaintext", sealed)
	}
	plain, err := b.Open(sealed)
	if err != nil || plain != "1//refresh-token" {
		t.Errorf("Open = %q, %v; want the plaintext back", plain, err)
	}
}

func TestSealingTwiceGivesDifferentCiphertexts(t *testing.T) {
	b := newBox(t, testKey(t))
	a, _ := b.Seal("same")
	c, _ := b.Seal("same")
	if a == c {
		t.Errorf("two seals of one plaintext are identical (%q): the nonce is not fresh", a)
	}
}

func TestOpenRejectsATamperedCiphertext(t *testing.T) {
	b := newBox(t, testKey(t))
	sealed, _ := b.Seal("1//refresh-token")
	raw, _ := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(sealed, "v1:"))
	raw[len(raw)-1] ^= 0x01
	tampered := "v1:" + base64.RawURLEncoding.EncodeToString(raw)
	if _, err := b.Open(tampered); !errors.Is(err, ErrOpen) {
		t.Errorf("Open(tampered) err = %v, want ErrOpen", err)
	}
}

func TestOpenRejectsAnotherKey(t *testing.T) {
	sealed, _ := newBox(t, testKey(t)).Seal("1//refresh-token")
	other := testKey(t)
	other[0] ^= 0xff
	if _, err := newBox(t, other).Open(sealed); !errors.Is(err, ErrOpen) {
		t.Errorf("Open with another key err = %v, want ErrOpen", err)
	}
}

func TestOpenRefusesAPlaintextRow(t *testing.T) {
	b := newBox(t, testKey(t))
	for _, legacy := range []string{"1//refresh-token", "", "v2:abc", "V1:abc"} {
		if _, err := b.Open(legacy); !errors.Is(err, ErrNotSealed) {
			t.Errorf("Open(%q) err = %v, want ErrNotSealed — a pre-encryption row must be refused, never guessed at", legacy, err)
		}
	}
	if _, err := b.Open("v1:not-base64!!"); !errors.Is(err, ErrOpen) {
		t.Errorf("Open(malformed v1) err = %v, want ErrOpen", err)
	}
}

func TestNewRejectsTheWrongKeyLength(t *testing.T) {
	for _, n := range []int{0, 16, 31, 33, 64} {
		if _, err := New(make([]byte, n)); !errors.Is(err, ErrBadKey) {
			t.Errorf("New(%d bytes) err = %v, want ErrBadKey", n, err)
		}
	}
}

func TestParseHexKeyBoundaries(t *testing.T) {
	ok := strings.Repeat("ab", 32)
	key, err := ParseHexKey(" " + ok + "\n")
	if err != nil || len(key) != KeyBytes {
		t.Fatalf("ParseHexKey(64 hex) = %d bytes, %v; want 32, nil (whitespace trimmed)", len(key), err)
	}
	for _, bad := range []string{"", strings.Repeat("ab", 31), strings.Repeat("ab", 33), strings.Repeat("zz", 32), "not hex at all"} {
		if _, err := ParseHexKey(bad); !errors.Is(err, ErrBadKey) {
			t.Errorf("ParseHexKey(%q) err = %v, want ErrBadKey", bad, err)
		}
	}
}
