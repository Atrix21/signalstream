package auth

import (
	"strings"
	"testing"
)

func TestEncryptor_Roundtrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "hello world"},
		{"empty string", ""},
		{"api key", "sk-abc123def456"},
		{"special chars", "p@$$w0rd!#%&*()+"},
		{"long text", strings.Repeat("a", 10000)},
		{"unicode", "emoji test"},
	}

	enc, err := NewEncryptor("12345678901234567890123456789012") // 32 bytes
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tt.plaintext)
			if err != nil {
				t.Fatalf("encrypt failed: %v", err)
			}

			if encrypted == tt.plaintext && tt.plaintext != "" {
				t.Fatal("encrypted text should differ from plaintext")
			}

			decrypted, err := enc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("decrypt failed: %v", err)
			}

			if decrypted != tt.plaintext {
				t.Fatalf("roundtrip mismatch: got %q, want %q", decrypted, tt.plaintext)
			}
		})
	}
}

func TestEncryptor_DifferentCiphertexts(t *testing.T) {
	enc, err := NewEncryptor("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	plain := "same-plaintext"
	a, _ := enc.Encrypt(plain)
	b, _ := enc.Encrypt(plain)

	if a == b {
		t.Fatal("encrypting the same plaintext twice should produce different ciphertexts (random nonce)")
	}

	// Both should decrypt correctly.
	da, _ := enc.Decrypt(a)
	db, _ := enc.Decrypt(b)
	if da != plain || db != plain {
		t.Fatal("both ciphertexts should decrypt to the same plaintext")
	}
}

func TestEncryptor_WrongKeyFails(t *testing.T) {
	enc1, _ := NewEncryptor("12345678901234567890123456789012")
	enc2, _ := NewEncryptor("abcdefghijklmnopqrstuvwxyz123456")

	encrypted, err := enc1.Encrypt("secret data")
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	_, err = enc2.Decrypt(encrypted)
	if err == nil {
		t.Fatal("decrypting with wrong key should fail")
	}
}

func TestEncryptor_InvalidKeyLength(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{"too short", "short"},
		{"31 bytes", "1234567890123456789012345678901"},
		{"33 bytes", "123456789012345678901234567890123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncryptor(tt.key)
			if err == nil {
				t.Fatalf("expected error for key length %d", len(tt.key))
			}
		})
	}
}

func TestEncryptor_TamperedCiphertext(t *testing.T) {
	enc, _ := NewEncryptor("12345678901234567890123456789012")

	encrypted, _ := enc.Encrypt("important data")

	// Tamper with the ciphertext.
	tampered := encrypted[:len(encrypted)-2] + "XX"
	_, err := enc.Decrypt(tampered)
	if err == nil {
		t.Fatal("tampered ciphertext should fail decryption")
	}
}
