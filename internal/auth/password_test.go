package auth

import "testing"

func TestPassword_HashAndVerify(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"simple", "password123"},
		{"complex", "C0mpl3x!P@$$w0rd#2024"},
		{"unicode", "p4ssw0rd"},
		{"long", "a-very-long-password-that-is-still-valid-for-bcrypt-hashing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := HashPassword(tt.password)
			if err != nil {
				t.Fatalf("hashing failed: %v", err)
			}

			if hash == tt.password {
				t.Fatal("hash should not equal plaintext")
			}

			if !CheckPasswordHash(tt.password, hash) {
				t.Fatal("correct password should verify")
			}
		})
	}
}

func TestPassword_WrongPasswordFails(t *testing.T) {
	hash, _ := HashPassword("correct-password")

	if CheckPasswordHash("wrong-password", hash) {
		t.Fatal("wrong password should not verify")
	}
}

func TestPassword_DifferentHashesPerCall(t *testing.T) {
	password := "same-password"
	hash1, _ := HashPassword(password)
	hash2, _ := HashPassword(password)

	if hash1 == hash2 {
		t.Fatal("bcrypt should produce different hashes for the same password (random salt)")
	}

	// Both should verify.
	if !CheckPasswordHash(password, hash1) || !CheckPasswordHash(password, hash2) {
		t.Fatal("both hashes should verify for the same password")
	}
}
