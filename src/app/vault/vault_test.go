package vault

import (
	"encoding/hex"
	"strings"
	"testing"
)

func newTestVault(t *testing.T) *Vault {
	t.Helper()
	key, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	v, err := New(key)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return v
}

func TestEncryptDecryptRoundTrip(t *testing.T) {
	v := newTestVault(t)
	cases := []string{"", "x", "hunter2", strings.Repeat("A", 1024), "🔐 unicode/emoji"}
	for _, plain := range cases {
		ct, err := v.Encrypt(plain)
		if err != nil {
			t.Fatalf("Encrypt(%q): %v", plain, err)
		}
		got, err := v.Decrypt(ct)
		if err != nil {
			t.Fatalf("Decrypt(%q): %v", plain, err)
		}
		if got != plain {
			t.Errorf("round-trip mismatch: got %q, want %q", got, plain)
		}
	}
}

func TestEmptyStringEncryptsToEmpty(t *testing.T) {
	v := newTestVault(t)
	ct, err := v.Encrypt("")
	if err != nil || ct != "" {
		t.Fatalf("Encrypt(\"\") = %q, %v; want \"\", nil", ct, err)
	}
	pt, err := v.Decrypt("")
	if err != nil || pt != "" {
		t.Fatalf("Decrypt(\"\") = %q, %v; want \"\", nil", pt, err)
	}
}

func TestEncryptIsNondeterministic(t *testing.T) {
	v := newTestVault(t)
	a, _ := v.Encrypt("same plaintext")
	b, _ := v.Encrypt("same plaintext")
	if a == b {
		t.Errorf("expected fresh nonce per encrypt; got identical ciphertexts %q", a)
	}
}

func TestDecryptWithWrongKeyFails(t *testing.T) {
	v1 := newTestVault(t)
	v2 := newTestVault(t)
	ct, err := v1.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := v2.Decrypt(ct); err == nil {
		t.Errorf("Decrypt with wrong key should fail, got nil error")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	v := newTestVault(t)
	ct, err := v.Encrypt("secret")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	raw, _ := hex.DecodeString(ct)
	raw[len(raw)-1] ^= 0x01
	tampered := hex.EncodeToString(raw)
	if _, err := v.Decrypt(tampered); err == nil {
		t.Errorf("Decrypt of tampered ciphertext should fail, got nil error")
	}
}

func TestDecryptInvalidHexFails(t *testing.T) {
	v := newTestVault(t)
	if _, err := v.Decrypt("not-hex-zz"); err == nil {
		t.Errorf("Decrypt of non-hex should fail")
	}
}

func TestDecryptTooShortFails(t *testing.T) {
	v := newTestVault(t)
	if _, err := v.Decrypt("0011"); err == nil {
		t.Errorf("Decrypt of short ciphertext should fail")
	}
}

func TestNewRejectsBadKey(t *testing.T) {
	if _, err := New("zz"); err == nil {
		t.Errorf("New should reject non-hex key")
	}
	if _, err := New(strings.Repeat("00", 17)); err == nil {
		t.Errorf("New should reject 17-byte key")
	}
}

func TestNewAcceptsAllAESKeySizes(t *testing.T) {
	for _, n := range []int{16, 24, 32} {
		if _, err := New(strings.Repeat("00", n)); err != nil {
			t.Errorf("New rejected valid %d-byte key: %v", n, err)
		}
	}
}

func TestGenerateKeyProduces256BitHex(t *testing.T) {
	k, err := GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey: %v", err)
	}
	if len(k) != 64 {
		t.Errorf("expected 64 hex chars (32 bytes), got %d", len(k))
	}
}
