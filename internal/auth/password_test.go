package auth

import "testing"

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("s3cret-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "s3cret-password" {
		t.Fatal("hash must not equal the plain password")
	}
	if !CheckPasswordHash("s3cret-password", hash) {
		t.Error("the original password should match its hash")
	}
	if CheckPasswordHash("wrong-password", hash) {
		t.Error("a wrong password must not match the hash")
	}
}

func TestHashPasswordIsSalted(t *testing.T) {
	first, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	second, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if first == second {
		t.Error("two hashes of the same password must differ (salt)")
	}
}
