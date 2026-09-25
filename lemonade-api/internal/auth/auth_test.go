package auth

import (
	"context"
	"errors"
	"testing"
)

func TestIdentityAcceptable(t *testing.T) {
	ok := Identity{UID: "u1", Email: "a@b.c", EmailVerified: true, Provider: "google.com"}
	if err := ok.Acceptable(); err != nil {
		t.Fatalf("valid identity rejected: %v", err)
	}
	for name, id := range map[string]Identity{
		"no uid":           {Email: "a@b.c", EmailVerified: true, Provider: "google.com"},
		"other provider":   {UID: "u1", Email: "a@b.c", EmailVerified: true, Provider: "password"},
		"unverified email": {UID: "u1", Email: "a@b.c", EmailVerified: false, Provider: "google.com"},
	} {
		if err := id.Acceptable(); !errors.Is(err, ErrInvalidToken) {
			t.Errorf("%s: err = %v, want ErrInvalidToken", name, err)
		}
	}
}

func TestFake(t *testing.T) {
	f := NewFake()
	id := Identity{UID: "u1", Provider: "google.com", EmailVerified: true}
	f.Add("tok", id)
	if got, err := f.Verify(context.Background(), "tok"); err != nil || got != id {
		t.Fatalf("Verify = %+v, %v", got, err)
	}
	if _, err := f.Verify(context.Background(), "nope"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("unknown token: %v", err)
	}
}

// A malformed token is rejected locally, with no network call.
func TestFirebaseRejectsGarbage(t *testing.T) {
	v, err := NewFirebase(context.Background(), "demo-lemonade")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(context.Background(), "not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("err = %v, want ErrInvalidToken", err)
	}
}
