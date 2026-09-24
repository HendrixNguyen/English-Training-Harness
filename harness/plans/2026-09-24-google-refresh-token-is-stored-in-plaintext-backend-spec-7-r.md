---
idea: harness/ideas/_inbox/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md
status: done
priority: high
merged: true
branch: harness/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r
worktree: .worktrees/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r
pr: "https://github.com/HendrixNguyen/English-Training-Harness/pull/17"
---
# Secrets at rest and at boot: AES-256-GCM for `users.google_refresh_token`, and a `JWT_SECRET` that must be 32 bytes — Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Idea (head):** `harness/ideas/_inbox/google-refresh-token-is-stored-in-plaintext-backend-spec-7-r.md`
**Also planned here (each idea's frontmatter points at this plan):**
- `harness/ideas/_inbox/jwt-secret-is-accepted-at-any-length-including-one-character.md` (folds the rejected `jwt-verify-does-not-require-exp-or-bind-iss-aud-and-bearer-i.md`) → Task 2 (length), Task 3 (`WithExpirationRequired`, case-insensitive `Bearer`)

**Goal:** The most sensitive column the product holds — standing write access to a user's Google Calendar and Tasks — is sealed with AES-256-GCM under `ENCRYPTION_SECRET_KEY` before it reaches Postgres and opened only by the google sync's token source, as backend spec §6.1, §7 and §9 require; and the process refuses to boot with an HS256 secret shorter than the hash it signs with.

**Why now (`priority: high`):** `backend/internal/auth/repo.go` `upsertUserSQL` stores `$4` verbatim and `backend/internal/google/token.go` `PgRefreshTokenSource` reads it verbatim; `grep -rn 'aes\|ENCRYPTION' backend/` finds nothing. Any dump, backup or log line exposes every user's refresh token. The fix gets strictly more expensive after the first real rows exist, and `JWT_SECRET=x` boots today (`config.Load` is presence-only) for a variable the spec never lists, so whoever provisions Railway types it by hand.

**Root cause (from the idea's `## Evaluation`):** the auth slice was written against the 1st-thinking doc only, before the backend spec existed; nothing in `config`, `auth` or `google` ever mentions a cipher.

**Design decisions (read before the tasks):**
1. **One new package, `backend/internal/secrets`,** owns the cipher: `New(key)`, `Seal`, `Open`, `ParseHexKey`. `auth` and `google` each see one small interface (`auth.Sealer`, `google.Opener`) and never the key; `cmd/api` builds one `*secrets.Box` and hands it to both. Packages still talk via interfaces, never each other's tables (CODEMAP rule).
2. **Seal in the repository, not the service.** Encryption at rest is a persistence concern: `auth.NewPgUserRepo(pool, sealer)` seals a non-empty token before the upsert, so `Service.SignIn`, its fakes and its tests are untouched, and the existing rule "an empty token keeps the stored value" (`COALESCE(NULLIF(…, ''), …)`) is preserved by passing `""` through unsealed.
3. **Ciphertext format** is `v1:` + base64url(nonce ‖ ciphertext‖tag), 12-byte random nonce per call. The prefix lets `Open` refuse a pre-encryption plaintext row deterministically (`ErrNotSealed`) instead of feeding it to GCM.
4. **Cutover without a data migration.** `google.PgRefreshTokenSource` maps an empty, unsealed or undecryptable stored value to `ErrNoRefreshToken`, which `Service.Sync` already turns into `ErrReauthRequired` → `409 reauth_required`; the PWA sends the user back through Google consent, and because `auth.Scopes` always sets `prompt=consent`, the next sign-in stores a fresh sealed token. Existing rows heal themselves on the user's next sign-in; the product has no launched users. A re-encryption migration would need the key inside a migration step and is deliberately not built (YAGNI).
5. **`ENCRYPTION_SECRET_KEY` is required** (spec §9 lists it): 64 hex characters → 32 bytes, checked in `config.Load` with a generate hint. **`JWT_SECRET` must be ≥ 32 bytes** (RFC 7518 §3.2: an HS256 key no shorter than the 256-bit hash output), same place, same style. Both refusals happen before any service is dialled.
6. **From the folded JWT-verify finding** this plan takes `jwt.WithExpirationRequired()` (a token without `exp` must never verify) and a case-insensitive `Bearer` scheme (RFC 7235 says the scheme is case-insensitive). `iss`/`aud` binding is **deliberately left out**: one issuer, one audience, one secret — a token this API mints can only be consumed by this API, and forging one already requires the secret; the claims would add bytes to every request and nothing to the threat model.

**Tech stack:** Go 1.25 stdlib `crypto/aes`, `crypto/cipher`, `crypto/rand`, `encoding/hex`, `encoding/base64`; `golang-jwt/jwt/v5` (already a dependency). No new dependencies.

**⚠ Merge-order / conflict note — read before `git worktree add`:** the approved cmd/api shutdown plan and today's API-edge plan both edit `config.go`, `config_test.go`, `main.go` and `.env.example`. Every edit below is a *region edit* (append a field, append a check before `return cfg, nil`, change one constructor argument, append a `.env.example` block). Suggested daily merge order: shutdown plan → API-edge plan → this plan. If `origin/main` moves, `git fetch origin main && git merge origin/main --no-edit`, keep both sides' additions, and re-run Task 2's step that updates *every* `JWT_SECRET` value in `config_test.go` — including tests the other plans added.

**Run every command from `backend/` inside the worktree** unless a step says otherwise. `rg` and `timeout` are not installed.

---

## File structure

| Path | Change |
| --- | --- |
| `backend/internal/secrets/secrets.go` | **New**: `KeyBytes`, `ErrBadKey`, `ErrNotSealed`, `ErrOpen`, `ParseHexKey`, `Box`, `New`, `Seal`, `Open` |
| `backend/internal/secrets/secrets_test.go` | **New**: round trip, nonce freshness, tamper, wrong key, plaintext refused, key length, hex parsing |
| `backend/internal/config/config.go` | `MinJWTSecretBytes`; `EncryptionKey []byte`; length check; required hex key |
| `backend/internal/config/config_test.go` | Every `JWT_SECRET` value → 32 bytes; every test sets `ENCRYPTION_SECRET_KEY`; boundary tables |
| `backend/internal/auth/repo.go` | `Sealer` interface; `PgUserRepo.sealer`; `NewPgUserRepo(pool, sealer)`; seal before the upsert |
| `backend/internal/auth/integration_test.go` | Assert the stored column is a `v1:` ciphertext that opens to the plaintext; the empty-token rule still holds |
| `backend/internal/auth/token.go` | `jwt.WithExpirationRequired()` |
| `backend/internal/auth/token_test.go` | `TestVerifyRejectsATokenWithoutExp` |
| `backend/internal/auth/middleware.go` | Case-insensitive `Bearer` |
| `backend/internal/auth/middleware_test.go` | `TestRequireAcceptsALowerCaseBearerScheme` |
| `backend/internal/google/token.go` | `Opener` interface; `PgRefreshTokenSource.opener`; `NewPgRefreshTokenSource(pool, opener)`; `openStored` |
| `backend/internal/google/token_test.go` | **New**: `openStored` cases with a real `secrets.Box` |
| `backend/internal/google/integration_test.go` | Seed a sealed token; legacy plaintext row → `ErrNoRefreshToken` |
| `backend/cmd/api/main.go` | Build the `Box`; pass to `auth.NewPgUserRepo` and `google.NewPgRefreshTokenSource` |
| `backend/.env.example` | `JWT_SECRET` / `ENCRYPTION_SECRET_KEY` block with generate hints |
| `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` | §7 bullet: restore the missing "AES-256-GCM"; §9 step 2: add `JWT\_SECRET` (≥ 32 bytes) |
| `project-base/1st-thinking-architecture-doc.md` | §8 env list: add `JWT\_SECRET` and `ENCRYPTION\_SECRET\_KEY` |
| `harness/CODEMAP.md` | New `**secrets**` bullet; `auth` and `google` paragraphs; `store`'s env note |

---

## Tasks

### Task 1: `internal/secrets` — AES-256-GCM seal/open with a versioned prefix

**Files:**
- Create: `backend/internal/secrets/secrets_test.go`
- Create: `backend/internal/secrets/secrets.go`

- [ ] **Step 1: Write the failing tests**

```go
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
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/secrets/ -count=1`
Expected: compile error — `undefined: New`, `KeyBytes`, …

- [ ] **Step 3: Implement `secrets.go`**

```go
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
```

- [ ] **Step 4: Run the tests**

Run: `go test ./internal/secrets/ -count=1 -v`
Expected: PASS ×7.

- [ ] **Step 5: Commit**

```bash
git add internal/secrets
git commit -m "secrets: AES-256-GCM box with a v1: prefix for users.google_refresh_token"
```

### Task 2: `config.Load` requires a 64-hex `ENCRYPTION_SECRET_KEY` and a ≥ 32-byte `JWT_SECRET`

**Files:**
- Modify: `backend/internal/config/config_test.go`
- Modify: `backend/internal/config/config.go`

- [ ] **Step 1: Update every existing test's secrets, then add the boundary tables**

`grep -n 's3cret' internal/config/config_test.go` lists seven `t.Setenv("JWT_SECRET", "s3cret")` lines (more if another plan added tests). Add two file-level constants and replace each of those lines with the pair below — **every** test that reaches `return cfg, nil` must now set both:

```go
// testJWTSecret is 32 bytes — the HS256 floor config enforces.
const testJWTSecret = "0123456789abcdef0123456789abcdef"

// testHexKey is 64 hex characters — a well-formed ENCRYPTION_SECRET_KEY.
const testHexKey = "00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff"
```

```go
	t.Setenv("JWT_SECRET", testJWTSecret)
	t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
```

In `TestLoadRequiresGoogleAndJWTSecrets`'s "all present" case, change `cfg.JWTSecret != "s3cret"` to `cfg.JWTSecret != testJWTSecret`. Then append:

```go
func TestLoadRejectsAShortJWTSecret(t *testing.T) {
	for _, tc := range []struct {
		n  int
		ok bool
	}{{1, false}, {31, false}, {32, true}, {64, true}} {
		t.Run(fmt.Sprintf("%d bytes", tc.n), func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
			t.Setenv("REDIS_URL", "redis://localhost:6379/0")
			t.Setenv("GOOGLE_CLIENT_ID", "cid")
			t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
			t.Setenv("ENCRYPTION_SECRET_KEY", testHexKey)
			t.Setenv("JWT_SECRET", strings.Repeat("x", tc.n))
			_, err := Load()
			if tc.ok && err != nil {
				t.Fatalf("Load() with a %d-byte JWT_SECRET = %v, want nil", tc.n, err)
			}
			if !tc.ok {
				if err == nil {
					t.Fatalf("Load() accepted a %d-byte JWT_SECRET; RFC 7518 §3.2 wants ≥ %d", tc.n, MinJWTSecretBytes)
				}
				if !strings.Contains(err.Error(), "JWT_SECRET must be at least 32 bytes") || !strings.Contains(err.Error(), "openssl rand -base64 32") {
					t.Errorf("error = %q, want the requirement and the generate hint", err)
				}
			}
		})
	}
}

func TestLoadRequiresAWellFormedEncryptionKey(t *testing.T) {
	for _, tc := range []struct {
		name, key, wantErr string
	}{
		{"unset", "", "ENCRYPTION_SECRET_KEY is required"},
		{"62 hex", strings.Repeat("ab", 31), "64 hex characters"},
		{"66 hex", strings.Repeat("ab", 33), "64 hex characters"},
		{"not hex", strings.Repeat("zz", 32), "64 hex characters"},
		{"64 hex", testHexKey, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
			t.Setenv("REDIS_URL", "redis://localhost:6379/0")
			t.Setenv("GOOGLE_CLIENT_ID", "cid")
			t.Setenv("GOOGLE_CLIENT_SECRET", "csecret")
			t.Setenv("JWT_SECRET", testJWTSecret)
			t.Setenv("ENCRYPTION_SECRET_KEY", tc.key)
			cfg, err := Load()
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Load() = %v, want nil", err)
				}
				if len(cfg.EncryptionKey) != 32 || cfg.EncryptionKey[0] != 0x00 || cfg.EncryptionKey[1] != 0x11 {
					t.Errorf("EncryptionKey = %x, want the 32 decoded bytes", cfg.EncryptionKey)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || !strings.Contains(err.Error(), "openssl rand -hex 32") {
				t.Errorf("Load() err = %v, want it to mention %q and the generate hint", err, tc.wantErr)
			}
		})
	}
}
```

(Add `"fmt"` and `"strings"` to the test imports.)

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/config/ -count=1`
Expected: `cfg.EncryptionKey undefined`, `undefined: MinJWTSecretBytes`.

- [ ] **Step 3: Implement in `config.go`**

Import `"github.com/HendrixNguyen/English-Training-Harness/backend/internal/secrets"`. Add:

```go
// MinJWTSecretBytes is the HS256 floor: RFC 7518 §3.2 requires a key at least
// as long as the hash output (256 bits). A shorter secret is brute-forceable
// offline from one captured token.
const MinJWTSecretBytes = 32
```

Extend `Config` (replace the `JWTSecret` comment; add the key field after it):

```go
	// JWTSecret signs session tokens (HS256) and must be ≥ MinJWTSecretBytes.
	// NOTE: JWT_SECRET is NOT in the 1st-thinking §8 list — see CODEMAP auth.
	JWTSecret string
	// EncryptionKey is the decoded ENCRYPTION_SECRET_KEY (backend spec §7/§9):
	// 32 raw bytes for AES-256-GCM over users.google_refresh_token. Required.
	EncryptionKey []byte
```

In `Load`, after the presence loop over `GOOGLE_CLIENT_ID`/`GOOGLE_CLIENT_SECRET`/`JWT_SECRET` and before the VAPID reads:

```go
	if len(cfg.JWTSecret) < MinJWTSecretBytes {
		return Config{}, fmt.Errorf("config: JWT_SECRET must be at least %d bytes (generate one with: openssl rand -base64 32)", MinJWTSecretBytes)
	}
	rawKey := os.Getenv("ENCRYPTION_SECRET_KEY")
	if rawKey == "" {
		return Config{}, fmt.Errorf("config: ENCRYPTION_SECRET_KEY is required — 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)")
	}
	key, err := secrets.ParseHexKey(rawKey)
	if err != nil {
		return Config{}, fmt.Errorf("config: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)")
	}
	cfg.EncryptionKey = key
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/config/ -count=1 -v`
Expected: every pre-existing test still PASS (each now sets both secrets) plus the two tables. `go vet ./...` clean.

- [ ] **Step 5: Commit**

```bash
git add internal/config
git commit -m "config: require ENCRYPTION_SECRET_KEY (64 hex) and a JWT_SECRET of at least 32 bytes"
```

### Task 3: `auth` seals the refresh token at write; `Verify` requires `exp`; `Bearer` is case-insensitive

**Files:**
- Modify: `backend/internal/auth/repo.go`
- Modify: `backend/internal/auth/integration_test.go`
- Modify: `backend/internal/auth/token_test.go`, `backend/internal/auth/token.go`
- Modify: `backend/internal/auth/middleware_test.go`, `backend/internal/auth/middleware.go`

- [ ] **Step 1: Write the failing tests**

`token_test.go` — append (import `"github.com/golang-jwt/jwt/v5"`):

```go
func TestVerifyRejectsATokenWithoutExp(t *testing.T) {
	// Same secret, valid signature, no exp claim: must not verify.
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{Subject: "user-1"}).SignedString([]byte("secret"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewTokenIssuer("secret", time.Now).Verify(tok); err == nil {
		t.Fatal("Verify accepted a token with no exp claim")
	}
}
```

`middleware_test.go` — append:

```go
func TestRequireAcceptsACaseInsensitiveBearerScheme(t *testing.T) {
	iss := NewTokenIssuer("secret", time.Now)
	tok, _ := iss.Issue("user-1")
	sess := newFakeSessions()
	_ = sess.Put(context.Background(), "user-1", tok, TokenTTL)
	r := newGuardedRouter(iss, sess)

	for _, header := range []string{"bearer " + tok, "BEARER " + tok, "Bearer " + tok} {
		if w := get(t, r, header); w.Code != http.StatusOK {
			t.Errorf("header %q → status %d, want 200 (RFC 7235: the scheme is case-insensitive)", header[:6], w.Code)
		}
	}
}
```

`integration_test.go` — after `repo := NewPgUserRepo(pg.Pool)` will no longer compile; rewrite that line and the final read-back:

```go
	box, err := secrets.New(bytes.Repeat([]byte{7}, secrets.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	repo := NewPgUserRepo(pg.Pool, box)
```

and replace the `if refresh != "rt-1"` check with:

```go
	// Backend spec §7: the column holds a v1: AES-256-GCM ciphertext, never the
	// plaintext; and the re-login that sent no token kept the sealed one.
	if refresh == "rt-1" || !strings.HasPrefix(refresh, "v1:") {
		t.Errorf("google_refresh_token = %q, want a sealed v1: value, not the plaintext", refresh)
	}
	if plain, err := box.Open(refresh); err != nil || plain != "rt-1" {
		t.Errorf("Open(stored) = %q, %v; want rt-1 — the stored token kept when Google sends none", plain, err)
	}
```

(imports: `"bytes"`, `"strings"`, and `…/internal/secrets`.)

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/auth/ -count=1`
Expected: `TestVerifyRejectsATokenWithoutExp` FAIL (accepted), `TestRequireAcceptsACaseInsensitiveBearerScheme` FAIL on `bearer`/`BEARER` (401), integration test: compile error on `NewPgUserRepo` arity.

- [ ] **Step 3: Implement**

`repo.go`:

```go
// Sealer encrypts a refresh token before it is stored (backend spec §6.1,
// §7: AES-256-GCM under ENCRYPTION_SECRET_KEY). *secrets.Box satisfies it.
type Sealer interface {
	Seal(plain string) (string, error)
}

// PgUserRepo is the real UserRepo. It seals google_refresh_token on the way
// in; google.PgRefreshTokenSource opens it on the way out.
type PgUserRepo struct {
	Pool   *pgxpool.Pool
	sealer Sealer
}

// NewPgUserRepo builds a repo over an existing pool and the process's sealer.
func NewPgUserRepo(pool *pgxpool.Pool, sealer Sealer) *PgUserRepo {
	return &PgUserRepo{Pool: pool, sealer: sealer}
}
```

and at the top of `UpsertByGoogleID`:

```go
	// Google omits the refresh token on silent re-consent; "" must reach the
	// SQL unsealed so COALESCE(NULLIF(…, ''), users.google_refresh_token)
	// keeps the stored (sealed) value. Anything else is sealed here — the
	// plaintext never reaches Postgres.
	stored := ""
	if refreshToken != "" {
		sealed, err := r.sealer.Seal(refreshToken)
		if err != nil {
			return User{}, fmt.Errorf("auth: sealing refresh token: %w", err)
		}
		stored = sealed
	}
```

and pass `stored` (not `refreshToken`) as the fourth `QueryRow` argument.

`token.go` — add `jwt.WithExpirationRequired(),` to the `ParseWithClaims` options and extend the `Verify` doc comment: "…and requires an `exp` claim".

`middleware.go`:

```go
// bearerToken extracts the credentials from an Authorization header whose
// scheme is "Bearer" in any case (RFC 7235 §2.1: schemes are case-insensitive).
func bearerToken(header string) string {
	const scheme = "bearer "
	if len(header) < len(scheme) || !strings.EqualFold(header[:len(scheme)], scheme) {
		return ""
	}
	return strings.TrimSpace(header[len(scheme):])
}
```

Update the `Require` doc comment's clause (a) to "verifies against `JWT_SECRET` and carries an `exp`".

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/auth/ -count=1 -v -run 'Exp|Bearer|Superseded|Malformed'`
Expected: PASS incl. `TestRequireRejectsAMissingOrMalformedHeader` (its `"Bearer"`-without-a-space case still 401s). Then `go test ./internal/auth/ -count=1` → ok (the integration test skips without `TEST_DATABASE_URL`).

- [ ] **Step 5: Commit**

```bash
git add internal/auth
git commit -m "auth: seal google_refresh_token at write; Verify requires exp; Bearer is case-insensitive"
```

### Task 4: `google` opens the sealed token; a legacy or undecryptable row is `reauth_required`

**Files:**
- Create: `backend/internal/google/token_test.go`
- Modify: `backend/internal/google/token.go`
- Modify: `backend/internal/google/integration_test.go`

- [ ] **Step 1: Write the failing tests**

`token_test.go`:

```go
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
```

`integration_test.go` — seed a sealed token and add the legacy case. Replace the `INSERT INTO users … google_refresh_token … "1//refresh"` argument with a sealed value and the source construction:

```go
	box, err := secrets.New(bytes.Repeat([]byte{3}, secrets.KeyBytes))
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := box.Seal("1//refresh")
	if err != nil {
		t.Fatal(err)
	}
	// … the INSERT passes `sealed` as $6 …

	// The refresh-token seam opens what auth sealed.
	tok, err := NewPgRefreshTokenSource(pg.Pool, box).RefreshToken(ctx, userID)
	if err != nil || tok != "1//refresh" {
		t.Errorf("RefreshToken = %q, %v; want the opened plaintext", tok, err)
	}
	// A pre-encryption row (plaintext, no v1: prefix) is "no token on file":
	// Service maps it to reauth_required and the next sign-in stores a sealed one.
	if _, err := pg.Pool.Exec(ctx, `UPDATE users SET google_refresh_token = '1//legacy-plaintext' WHERE id = $1`, userID); err != nil {
		t.Fatal(err)
	}
	if _, err := NewPgRefreshTokenSource(pg.Pool, box).RefreshToken(ctx, userID); !errors.Is(err, ErrNoRefreshToken) {
		t.Errorf("RefreshToken on a legacy plaintext row: err = %v, want ErrNoRefreshToken", err)
	}
```

(imports: `"bytes"`, `…/internal/secrets`.)

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/google/ -count=1`
Expected: compile errors — `undefined: openStored`, `NewPgRefreshTokenSource` arity.

- [ ] **Step 3: Implement in `token.go`**

Replace the type-comment block, struct, constructor and method:

```go
// ErrNoRefreshToken means there is no usable refresh token on file: the
// column is NULL/empty, or it holds a value this process cannot open (a
// pre-encryption plaintext row, a tampered value, another key). Service maps
// it to ErrReauthRequired: the PWA sends the user back through consent and
// auth stores a fresh sealed token — the cutover needs no data migration.
var ErrNoRefreshToken = errors.New("google: no refresh token on file")

// Opener decrypts what auth sealed (backend spec §7). *secrets.Box satisfies it.
type Opener interface {
	Open(sealed string) (string, error)
}

// RefreshTokenSource is the ONLY way this package reads
// users.google_refresh_token.
type RefreshTokenSource interface {
	RefreshToken(ctx context.Context, userID string) (string, error)
}

// PgRefreshTokenSource reads the sealed column and opens it.
type PgRefreshTokenSource struct {
	Pool   *pgxpool.Pool
	opener Opener
}

// NewPgRefreshTokenSource builds the source over an existing pool and the
// process's opener (the same Box auth seals with).
func NewPgRefreshTokenSource(pool *pgxpool.Pool, opener Opener) *PgRefreshTokenSource {
	return &PgRefreshTokenSource{Pool: pool, opener: opener}
}

const refreshTokenSQL = `SELECT COALESCE(google_refresh_token, '') FROM users WHERE id = $1`

func (s *PgRefreshTokenSource) RefreshToken(ctx context.Context, userID string) (string, error) {
	var stored string
	if err := s.Pool.QueryRow(ctx, refreshTokenSQL, userID).Scan(&stored); err != nil {
		return "", fmt.Errorf("google: reading refresh token: %w", err)
	}
	return openStored(s.opener, stored)
}

// openStored maps the raw column to a usable token or ErrNoRefreshToken.
func openStored(opener Opener, stored string) (string, error) {
	if stored == "" {
		return "", ErrNoRefreshToken
	}
	plain, err := opener.Open(stored)
	if err != nil {
		return "", fmt.Errorf("%w: stored value unusable (%v)", ErrNoRefreshToken, err)
	}
	return plain, nil
}
```

- [ ] **Step 4: Run the package tests**

Run: `go test ./internal/google/ -count=1 -v -run 'OpenStored|Reauth|NoRefresh'`
Expected: PASS (the existing `h.tokens.err = ErrNoRefreshToken` service test and the handler's 409 test are unchanged and still green). `go test ./internal/google/ -count=1` → ok.

- [ ] **Step 5: Commit**

```bash
git add internal/google
git commit -m "google: open the sealed refresh token; legacy or undecryptable rows are reauth_required"
```

### Task 5: Wire the box in `main.go`; `.env.example`, both specs, CODEMAP; boot refusals

**Files:**
- Modify: `backend/cmd/api/main.go` (one construction + two constructor arguments)
- Modify: `backend/.env.example`
- Modify: `project-base/Adaptive English Learning Platform - Backend Technical Specification.md` (§7 bullet, §9 step 2)
- Modify: `project-base/1st-thinking-architecture-doc.md` (§8 env list)
- Modify: `harness/CODEMAP.md`

- [ ] **Step 1: `main.go`** — import `…/internal/secrets`; after `cfg, err := config.Load()` succeeds (before `store.NewPostgres`):

```go
	// Backend spec §7: users.google_refresh_token is sealed with AES-256-GCM.
	// One Box: auth seals with it at sign-in, google opens with it at sync.
	box, err := secrets.New(cfg.EncryptionKey)
	if err != nil {
		log.Fatalf("secrets: %v", err)
	}
```

Change `auth.NewPgUserRepo(pg.Pool)` → `auth.NewPgUserRepo(pg.Pool, box)`, and `google.NewPgRefreshTokenSource(pg.Pool), // plaintext today; the §7 encryption fix replaces only this` → `google.NewPgRefreshTokenSource(pg.Pool, box), // opens what auth sealed (backend spec §7)`.

Run: `go build ./... && go vet ./...` → no output.

- [ ] **Step 2: `.env.example`** — append (or merge into the shutdown plan's application section if present, keeping these comments):

```
# Secrets (backend spec §7 / §9). Both are REQUIRED; boot refuses otherwise.
# JWT_SECRET signs HS256 session tokens — at least 32 bytes:
#   openssl rand -base64 32
# ENCRYPTION_SECRET_KEY seals users.google_refresh_token with AES-256-GCM —
# exactly 32 bytes as 64 hex characters:
#   openssl rand -hex 32
# Rotating or losing it makes every stored refresh token unreadable; affected
# users get 409 reauth_required on Google sync and re-consent at next sign-in.
#JWT_SECRET=
#ENCRYPTION_SECRET_KEY=
```

- [ ] **Step 3: Specs** (keep each document's backslash-escaping):
  - Backend spec §7, first bullet: `users.google\_refresh\_token encrypted with  via ENCRYPTION\_SECRET\_KEY (32-byte hex).` → `… encrypted with AES-256-GCM via ENCRYPTION\_SECRET\_KEY (32-byte hex).` (the algorithm name was lost in the paste — §6.1 names it).
  - Backend spec §9 step 2: after `ENCRYPTION\_SECRET\_KEY`, add `, JWT\_SECRET (at least 32 bytes)`.
  - 1st-thinking §8 "Environment Variables" list: add two bullets in the same style — `\`JWT\_SECRET\`: HS256 session-token secret, at least 32 bytes` and `\`ENCRYPTION\_SECRET\_KEY\`: 32-byte hex key for the refresh-token cipher`.

- [ ] **Step 4: CODEMAP**
  - New bullet after `**store**`: `- **secrets** — AES-256-GCM over one key (\`ENCRYPTION_SECRET_KEY\`, 64 hex → 32 bytes, required by \`config\`): \`Box.Seal\` → \`v1:\` + base64url(nonce‖ct) with a fresh nonce, \`Box.Open\` → \`ErrNotSealed\` for a value without the prefix (a pre-encryption row) or \`ErrOpen\` for anything that does not authenticate. Used only for \`users.google_refresh_token\` (backend spec §6.1/§7): \`auth.PgUserRepo\` seals at sign-in, \`google.PgRefreshTokenSource\` opens at sync; \`cmd/api\` builds the one \`Box\`. Pure tests.`
  - `**auth**`: after "upserts `users` on `google_id` (refreshes email/full_name/refresh token; …)" add "— the refresh token is **sealed** (`secrets.Box`, `v1:` prefix) before it reaches the upsert; an empty token still keeps the stored value —"; replace "Needs `JWT_SECRET`, which is **absent from the spec §8 env list**." with "Needs `JWT_SECRET` (≥ 32 bytes, enforced by `config.Load` — RFC 7518 §3.2) and `ENCRYPTION_SECRET_KEY`; both are now in backend spec §9 and 1st-thinking §8. `Verify` requires `exp`; the `Bearer` scheme is case-insensitive."
  - `**google**`: replace "read **only** through `google.RefreshTokenSource`, whose single implementation `PgRefreshTokenSource` returns the column as stored (plaintext; the §7 AES-256-GCM inbox bug replaces that one struct)" with "read **only** through `google.RefreshTokenSource`, whose single implementation `PgRefreshTokenSource` opens the sealed column with the process's `secrets.Box`; an empty, unsealed (pre-encryption) or undecryptable value is `ErrNoRefreshToken` → `409 reauth_required`, so a legacy row heals itself at the user's next sign-in (`prompt=consent` always returns a fresh token) — no data migration".
  - `**store**` bullet or the CI section: note that no integration test needs `ENCRYPTION_SECRET_KEY` — each builds its own `Box`.

- [ ] **Step 5: Boot refusals (no services needed — `config.Load` runs before any dial)**

```bash
JWT_SECRET=x ENCRYPTION_SECRET_KEY=$(openssl rand -hex 32) DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
# expect: config: JWT_SECRET must be at least 32 bytes (generate one with: openssl rand -base64 32)   exit status 1
JWT_SECRET=$(openssl rand -base64 32) DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
# expect: config: ENCRYPTION_SECRET_KEY is required — 32 bytes as 64 hex characters (…)   exit status 1
JWT_SECRET=$(openssl rand -base64 32) ENCRYPTION_SECRET_KEY=abc DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
# expect: config: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters (…)   exit status 1
```

Record the three lines in the execution summary.

- [ ] **Step 6: Commit**

```bash
git add cmd/api/main.go .env.example ../project-base ../harness/CODEMAP.md
git commit -m "cmd/api: one secrets box for auth and google; document JWT_SECRET and ENCRYPTION_SECRET_KEY"
```

---

## Verification

```bash
cd backend
gofmt -l ./internal/secrets ./internal/config ./internal/auth ./internal/google ./cmd/api
# expect: no output
go build ./... && go vet ./... && go test ./... -count=1
# expect: ok for every package, no live service needed
go test ./internal/secrets/ -count=1 -v
# expect: PASS ×7 — SealThenOpenRoundTrips, SealingTwiceGivesDifferentCiphertexts, OpenRejectsATamperedCiphertext,
#         OpenRejectsAnotherKey, OpenRefusesAPlaintextRow, NewRejectsTheWrongKeyLength, ParseHexKeyBoundaries
go test ./internal/config/ -count=1 -v -run 'JWT|Encryption'
# expect: PASS — RejectsAShortJWTSecret (1, 31 rejected; 32, 64 accepted), RequiresAWellFormedEncryptionKey (5 subtests)
go test ./internal/auth/ -count=1 -v -run 'WithoutExp|CaseInsensitiveBearer'
# expect: PASS ×2
go test ./internal/google/ -count=1 -v -run 'OpenStored'
# expect: PASS ×2
grep -n 'refreshToken, defaultTargetGoal' internal/auth/repo.go
# expect: no output — the SQL receives `stored`, never the plaintext argument
grep -c 'WithExpirationRequired' internal/auth/token.go
# expect: 1
grep -n 'secrets.New(cfg.EncryptionKey)\|NewPgUserRepo(pg.Pool, box)\|NewPgRefreshTokenSource(pg.Pool, box)' cmd/api/main.go
# expect: 3 lines
grep -rn 'plaintext today' cmd/api/main.go internal/google/token.go ../harness/CODEMAP.md
# expect: no output — the seam comment is gone everywhere
grep -c 'ENCRYPTION_SECRET_KEY' .env.example ../harness/CODEMAP.md; grep -c 'AES-256-GCM via ENCRYPTION' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md"; grep -c 'JWT\\_SECRET' "../project-base/Adaptive English Learning Platform - Backend Technical Specification.md" ../project-base/1st-thinking-architecture-doc.md
# expect: ≥ 1 each
git log --oneline origin/main..HEAD | wc -l
# expect: 5 commits, one per task, each with the Co-Authored-By trailer
python3 ../tools/harness/cli.py validate; echo "exit=$?"
# expect: exit=0

# With the dev stack (unique compose project) and TEST_DATABASE_URL exported:
COMPOSE_PROJECT_NAME=secrets POSTGRES_PORT=5462 REDIS_PORT=6412 docker compose up -d --wait --wait-timeout 60
export TEST_DATABASE_URL=postgres://english:english@localhost:5462/english?sslmode=disable TEST_REDIS_URL=redis://localhost:6412/0
go test ./internal/auth/ ./internal/google/ -count=1 -v -run Integration -p 1
# expect: PASS TestIntegrationUpsertCreatesThenPreservesTheLearnerState — stored value has the v1: prefix, is not "rt-1", Open() == "rt-1" after the empty re-login;
#         PASS TestIntegrationSyncStateIsOneRowPerUser — RefreshToken opens the seeded sealed token; a legacy plaintext row → ErrNoRefreshToken
docker compose -p secrets down
```

Mutation checks (each must turn the named test red, then restore):

| Mutation | Test that fails |
| --- | --- |
| `Seal`: constant nonce (`make([]byte, 12)` without `rand.Read`) | `SealingTwiceGivesDifferentCiphertexts` |
| `Open`: skip the prefix check | `OpenRefusesAPlaintextRow` |
| `config`: drop the `MinJWTSecretBytes` check | `RejectsAShortJWTSecret` (1, 31) |
| `auth/repo.go`: pass `refreshToken` instead of `stored` | integration `…PreservesTheLearnerState` (stored == "rt-1") |
| `auth/repo.go`: seal even when `refreshToken == ""` | integration `…PreservesTheLearnerState` (re-login with "" overwrites the stored token with a sealed empty string → `Open` ≠ "rt-1") |
| `token.go`: remove `WithExpirationRequired()` | `VerifyRejectsATokenWithoutExp` |
| `google/token.go`: return `stored` without `Open` | `OpenStoredReturnsThePlaintextOfASealedToken`, integration legacy case |

Boot refusals: Task 5 Step 5's three messages, recorded in the execution summary.

## Notes and open questions

- **Rotation** is out of scope: a key change turns every stored token into `reauth_required` until the user signs in again, which is acceptable pre-launch and documented in `.env.example`. A `v2:` prefix and a two-key `Open` are the shape of the future change; nothing here prevents it.
- **Why not encrypt in `Service.SignIn`?** It would put a persistence concern in the orchestration layer and change `UserRepo`'s contract for every fake. The repository is where the row is written; that is where at-rest encryption belongs.
- **`ENCRYPTION_SECRET_KEY` is required, `VAPID_*` are optional** — different postures on purpose: the app is fully functional without Web Push, but storing the refresh token unsealed is the defect this plan removes, so there is no unsealed mode.
- **CI:** `backend-integration` needs no new env — both integration tests build a `Box` from a literal key. `backend-unit` is unaffected.
- **Existing Railway rows** (if any test users exist): after deploy, their first `POST /integrations/google/sync` answers `409 reauth_required`; the PWA's settings screen (planned separately) routes that through consent. No operator action.

## Execution summary

Built exactly as planned, 5 commits (one per task), no deviations from the plan's design or file structure.

**Deviation (environment, not scope):** this session's write-tool sandbox is confined to paths under this session's own worktree directory. The worktree was therefore created at `.claude/worktrees/zen-burnell-b29ff5/.worktrees/<slug>` instead of the top-level `<repo>/.worktrees/<slug>` the harness convention uses elsewhere. It is still on `origin/main` and the correct `harness/*` branch; the `worktree:` frontmatter (`.worktrees/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r`, relative to ROOT) is accurate for ROOT's own bookkeeping. No plan content or task was changed.

### Verification (plan's Verification section)

```
$ gofmt -l ./internal/secrets ./internal/config ./internal/auth ./internal/google ./cmd/api
(no output)

$ go build ./... && go vet ./... && go test ./... -count=1
ok  .../internal/airouter, auth, config, google, health, notify, onboarding, pet, quests, secrets, store  (all ok, no live service needed)

$ go test ./internal/secrets/ -count=1 -v
PASS ×7: SealThenOpenRoundTrips, SealingTwiceGivesDifferentCiphertexts, OpenRejectsATamperedCiphertext,
         OpenRejectsAnotherKey, OpenRefusesAPlaintextRow, NewRejectsTheWrongKeyLength, ParseHexKeyBoundaries

$ go test ./internal/config/ -count=1 -v -run 'JWT|Encryption'
PASS: TestLoadRejectsAShortJWTSecret (1,31 bytes → err; 32,64 bytes → nil), TestLoadRequiresAWellFormedEncryptionKey (5 subtests)

$ go test ./internal/auth/ -count=1 -v -run 'WithoutExp|CaseInsensitiveBearer'
PASS ×2: TestVerifyRejectsATokenWithoutExp, TestRequireAcceptsACaseInsensitiveBearerScheme

$ go test ./internal/google/ -count=1 -v -run 'OpenStored'
PASS ×2: TestOpenStoredReturnsThePlaintextOfASealedToken, TestOpenStoredMapsEmptyLegacyAndTamperedToErrNoRefreshToken

$ grep -n 'refreshToken, defaultTargetGoal' internal/auth/repo.go   → (no output, as expected)
$ grep -c 'WithExpirationRequired' internal/auth/token.go            → 1
$ grep -n 'secrets.New(cfg.EncryptionKey)\|NewPgUserRepo(pg.Pool, box)\|NewPgRefreshTokenSource(pg.Pool, box)' cmd/api/main.go → 3 lines
$ grep -rn 'plaintext today' cmd/api/main.go internal/google/token.go ../harness/CODEMAP.md → (no output)
$ grep -c 'ENCRYPTION_SECRET_KEY' .env.example ../harness/CODEMAP.md → 2, 3
$ grep -c 'AES-256-GCM via ENCRYPTION' backend-spec.md → 1
$ grep -c 'JWT\_SECRET' backend-spec.md 1st-thinking-architecture-doc.md → 1, 1
$ git log --oneline origin/main..HEAD | wc -l → 5
$ python3 ../tools/harness/cli.py validate; echo exit=$? → exit=0

# Docker Compose (COMPOSE_PROJECT_NAME=exec-secrets, POSTGRES_PORT=55434, REDIS_PORT=56381) + TEST_DATABASE_URL/TEST_REDIS_URL:
$ go test ./internal/auth/ ./internal/google/ -count=1 -v -run Integration -p 1
PASS TestIntegrationUpsertCreatesThenPreservesTheLearnerState (stored value has v1: prefix, is not "rt-1", Open() == "rt-1" after the empty re-login)
PASS TestIntegrationSyncStateIsOneRowPerUser (RefreshToken opens the seeded sealed token; the legacy-plaintext row after it → ErrNoRefreshToken)
$ make test-integration  → every package's Integration tests PASS (airouter, auth, google, notify, onboarding, pet, quests, store) — nothing else broke
$ docker compose -p exec-secrets down → clean
```

### Boot refusals (Task 5 Step 5)

```
$ JWT_SECRET=x ENCRYPTION_SECRET_KEY=$(openssl rand -hex 32) DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
config: JWT_SECRET must be at least 32 bytes (generate one with: openssl rand -base64 32)   exit status 1

$ JWT_SECRET=$(openssl rand -base64 32) DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
config: ENCRYPTION_SECRET_KEY is required — 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)   exit status 1

$ JWT_SECRET=$(openssl rand -base64 32) ENCRYPTION_SECRET_KEY=abc DATABASE_URL=postgres://x REDIS_URL=redis://x GOOGLE_CLIENT_ID=x GOOGLE_CLIENT_SECRET=x go run ./cmd/api
config: ENCRYPTION_SECRET_KEY must be 32 bytes as 64 hex characters (generate one with: openssl rand -hex 32)   exit status 1
```

### Runtime proof (skill step 8)

- **Build:** `go build ./...` clean, `go vet ./...` clean.
- **Whole suite, clean shell** (no DATABASE_URL/REDIS_URL/JWT_SECRET/ENCRYPTION_SECRET_KEY/GOOGLE_CLIENT_* set): `go test ./... -count=1` → every package `ok`.
- **Boots and answers a real path:** with a real 32-byte `JWT_SECRET`, a real 64-hex `ENCRYPTION_SECRET_KEY`, and the isolated Postgres/Redis (ports 55434/56381) up, `go run ./cmd/api` on port 18083 logged migrations applied, mounted every route, and `curl http://localhost:18083/healthz` → `200 {"postgres":"ok","redis":"ok","status":"ok"}`. Exercising the actual sign-in/sync seal-then-open path end-to-end needs a real Google OAuth exchange (the codebase deliberately never calls Google in tests — see CODEMAP), so the seal/open code paths added by this plan are proven against the same live Postgres via the two Integration tests above (`auth` seals at upsert, `google` opens at read, legacy plaintext → `ErrNoRefreshToken`), run through the real `cmd/api` wiring's constructors.
- **Boot refusals:** verified above — short `JWT_SECRET`, missing `ENCRYPTION_SECRET_KEY`, malformed `ENCRYPTION_SECRET_KEY` all refuse before any service dial, with the documented generate hints.
- **Documented commands:** `make test`, `make up`/`make down` (via `docker compose up -d --wait`/`down -p exec-secrets`), `make test-integration` all ran as documented, with a unique `COMPOSE_PROJECT_NAME`/ports so no other worktree's containers were touched.
- **Cleanup:** `go run` process and its compiled child were killed (verified via `lsof -i :18083` and `pgrep`, both empty afterward); `docker compose -p exec-secrets down` removed both containers and the network; scratch `backend/.env` deleted. `git status` in the worktree is clean.
- **CI:** pushed `harness/2026-09-24-high-google-refresh-token-is-stored-in-plaintext-backend-spec-7-r`; run [35958815050](https://github.com/HendrixNguyen/English-Training-Harness/actions/runs/35958815050) — **success** (`backend-unit`, `backend-integration`, `harness-tooling`, `frontend` all green).
