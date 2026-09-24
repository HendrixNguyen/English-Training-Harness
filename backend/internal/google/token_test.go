package google

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/HendrixNguyen/English-Training-Harness/backend/internal/secrets"
)

func testBox(t *testing.T) *secrets.Box {
	t.Helper()
	b, err := secrets.New(bytes.Repeat([]byte{9}, secrets.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestOpenStoredReturnsThePlaintextOfASealedToken(t *testing.T) {
	box := testBox(t)
	sealed, _ := box.Seal("1//refresh")
	got, err := openStored(box, sealed)
	if err != nil || got != "1//refresh" {
		t.Errorf("openStored = %q, %v; want 1//refresh", got, err)
	}
}

func TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken(t *testing.T) {
	box := testBox(t)
	sealed, _ := box.Seal("1//refresh")
	tampered := sealed[:len(sealed)-2] + "AA"
	for name, stored := range map[string]string{"empty": "", "legacy plaintext": "1//refresh", "tampered": tampered} {
		_, err := openStored(box, stored)
		if !errors.Is(err, ErrNoRefreshToken) {
			t.Errorf("%s: err = %v, want ErrNoRefreshToken (→ 409 reauth_required, the self-healing cutover)", name, err)
		}
	}
	_, err := openStored(box, "1//refresh")
	if !strings.Contains(err.Error(), "not a v1 ciphertext") {
		t.Errorf("legacy error = %v, want it to say why (the operator reads this in the sync log)", err)
	}
}
