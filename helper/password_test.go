package helper

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	pass := "admin123"
	hash, err := HashPassword(pass)
	if err != nil {
		t.Fatalf("hash err: %v", err)
	}
	ok := CheckPasswordHash(pass, hash)
	if !ok {
		t.Fatalf("password did not match hash")
	}
	// negative check
	if CheckPasswordHash("wrong", hash) {
		t.Fatalf("expected mismatch for wrong password")
	}
}
