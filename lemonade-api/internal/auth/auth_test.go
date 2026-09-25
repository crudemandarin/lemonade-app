package auth

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr string // substring; empty means valid
	}{
		{"firebase with project", Config{Mode: ModeFirebase, FirebaseProjectID: "demo"}, ""},
		{"firebase is the default mode", Config{FirebaseProjectID: "demo"}, ""},
		{"firebase without project", Config{Mode: ModeFirebase}, "FIREBASE_PROJECT_ID"},
		{"dev locally", Config{Mode: ModeDev}, ""},
		{"dev on Cloud Run is refused", Config{Mode: ModeDev, OnCloudRun: true}, "AUTH_MODE=dev"},
		{"dev in production is refused", Config{Mode: ModeDev, AppEnv: "production"}, "AUTH_MODE=dev"},
		{"dev in Production, any case, is refused", Config{Mode: ModeDev, AppEnv: "Production"}, "AUTH_MODE=dev"},
		{"unknown mode", Config{Mode: "both", FirebaseProjectID: "demo"}, "AUTH_MODE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Fatalf("error = %v, want it to mention %q", err, tc.wantErr)
			}
		})
	}
}

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
