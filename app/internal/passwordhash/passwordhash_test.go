package passwordhash

import (
	"strings"
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword() returned error: %v", err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$") {
		t.Fatalf("HashPassword() = %q, want an Argon2id PHC string", hash)
	}
	if hash == "correct horse battery staple" {
		t.Fatal("HashPassword() returned the plaintext password")
	}
	if !ComparePassword(hash, "correct horse battery staple") {
		t.Fatal("ComparePassword() rejected the correct password")
	}
	if ComparePassword(hash, "incorrect") {
		t.Fatal("ComparePassword() accepted an incorrect password")
	}
}

func TestHashPasswordUsesAUniqueSalt(t *testing.T) {
	first, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("first HashPassword() returned error: %v", err)
	}
	second, err := HashPassword("same password")
	if err != nil {
		t.Fatalf("second HashPassword() returned error: %v", err)
	}
	if first == second {
		t.Fatal("HashPassword() returned the same hash for two calls")
	}
}

func TestComparePasswordRejectsMalformedHash(t *testing.T) {
	for _, encodedHash := range []string{
		"secret",
		"$2b$12$not-an-argon2-hash",
		"$argon2id$v=19$m=1,t=1,p=1$invalid$invalid",
	} {
		if ComparePassword(encodedHash, "secret") {
			t.Errorf("ComparePassword(%q) accepted a malformed hash", encodedHash)
		}
		if IsEncoded(encodedHash) {
			t.Errorf("IsEncoded(%q) accepted a malformed hash", encodedHash)
		}
	}
}
