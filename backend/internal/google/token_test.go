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
	got, err := openStored(box, "u1", sealed)
	if err != nil || got != "1//refresh" {
		t.Errorf("openStored = %q, %v; want 1//refresh", got, err)
	}
}

func TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken(t *testing.T) {
	box := testBox(t)
	sealed, _ := box.Seal("1//refresh")
	tampered := sealed[:len(sealed)-2] + "AA"
	for name, stored := range map[string]string{"empty": "", "legacy plaintext": "1//refresh", "tampered": tampered} {
		_, err := openStored(box, "u1", stored)
		if !errors.Is(err, ErrNoRefreshToken) {
			t.Errorf("%s: err = %v, want ErrNoRefreshToken (→ 409 reauth_required, the self-healing cutover)", name, err)
		}
	}
	_, err := openStored(box, "u1", "1//refresh")
	if !strings.Contains(err.Error(), "not a v1 ciphertext") {
		t.Errorf("legacy error = %v, want it to say why (the operator reads this in the sync log)", err)
	}
}

// captureLog is the package's helper from handler_test.go.

func TestOpenStoredLogsAnUndecryptableTokenWithTheUserIDAndNoValue(t *testing.T) {
	box := testBox(t)
	sealed, _ := box.Seal("1//refresh")
	otherKey, err := secrets.New(bytes.Repeat([]byte{7}, secrets.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]struct {
		opener Opener
		stored string
	}{
		"wrong key": {otherKey, sealed},
		"tampered":  {box, sealed[:len(sealed)-2] + "AA"},
	}
	for name, c := range cases {
		buf := captureLog(t)
		_, err := openStored(c.opener, "u1", c.stored)
		if !errors.Is(err, ErrNoRefreshToken) {
			t.Errorf("%s: err = %v, want ErrNoRefreshToken (409 reauth_required unchanged)", name, err)
		}
		if !errors.Is(err, secrets.ErrOpen) {
			t.Errorf("%s: err = %v, want it to wrap secrets.ErrOpen so a wrong key is distinguishable", name, err)
		}
		logged := buf.String()
		for _, want := range []string{"google: refresh token for user=u1 did not decrypt", "ciphertext did not authenticate"} {
			if !strings.Contains(logged, want) {
				t.Errorf("%s: log %q, want it to contain %q", name, logged, want)
			}
		}
		for _, secret := range []string{c.stored, "1//refresh"} {
			if strings.Contains(logged, secret) {
				t.Errorf("%s: log %q leaks the stored value %q", name, logged, secret)
			}
		}
	}
}

func TestOpenStoredNotesALegacyRowAndStaysQuietForAnEmptyColumn(t *testing.T) {
	box := testBox(t)

	buf := captureLog(t)
	_, err := openStored(box, "u1", "1//refresh")
	if !errors.Is(err, secrets.ErrNotSealed) {
		t.Errorf("legacy: err = %v, want it to wrap secrets.ErrNotSealed", err)
	}
	logged := buf.String()
	if !strings.Contains(logged, "user=u1") || !strings.Contains(logged, "not a v1 ciphertext") {
		t.Errorf("legacy: log %q, want user=u1 and the ErrNotSealed reason", logged)
	}
	if strings.Contains(logged, "1//refresh") {
		t.Errorf("legacy: log %q leaks the plaintext row", logged)
	}

	buf = captureLog(t)
	_, err = openStored(box, "u1", "")
	if err != ErrNoRefreshToken {
		t.Errorf("empty: err = %v, want exactly ErrNoRefreshToken", err)
	}
	if buf.Len() != 0 {
		t.Errorf("empty: logged %q, want nothing (a missing token is normal re-consent traffic)", buf.String())
	}
}
