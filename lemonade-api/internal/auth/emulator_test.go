package auth

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

// unsignedJWT builds the fake Google id_token the emulator accepts as an IdP credential.
func unsignedJWT(claims map[string]any) string {
	enc := func(v any) string {
		b, _ := json.Marshal(v)
		return base64.RawURLEncoding.EncodeToString(b)
	}
	return enc(map[string]string{"alg": "none", "typ": "JWT"}) + "." + enc(claims) + "."
}

// Runs a real token through the Firebase Auth Emulator and our verifier. Skipped
// unless FIREBASE_AUTH_EMULATOR_HOST is set (see the auth profile in docker-compose.yml).
func TestFirebaseVerifiesEmulatorTokens(t *testing.T) {
	host := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if host == "" {
		t.Skip("FIREBASE_AUTH_EMULATOR_HOST not set")
	}
	const project = "demo-lemonade"

	sign := func() string {
		body, _ := json.Marshal(map[string]any{
			"postBody":            "providerId=google.com&id_token=" + unsignedJWT(map[string]any{"sub": "g-123", "email": "player@example.com", "email_verified": true}),
			"requestUri":          "http://localhost",
			"returnIdpCredential": true,
			"returnSecureToken":   true,
		})
		resp, err := http.Post("http://"+host+"/identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=fake", "application/json", bytes.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out struct {
			IDToken string `json:"idToken"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.IDToken == "" {
			t.Fatalf("emulator sign-in: status %d, err %v", resp.StatusCode, err)
		}
		return out.IDToken
	}

	v, err := NewFirebase(context.Background(), project)
	if err != nil {
		t.Fatal(err)
	}
	id, err := v.Verify(context.Background(), sign())
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if id.UID == "" || id.Provider != "google.com" || id.Email != "player@example.com" || !id.EmailVerified {
		t.Fatalf("identity = %+v", id)
	}
	if err := id.Acceptable(); err != nil {
		t.Fatal(err)
	}

	// A token for another project is refused (audience check).
	other, _ := NewFirebase(context.Background(), "some-other-project")
	if _, err := other.Verify(context.Background(), sign()); err == nil {
		t.Fatal("token for a different project accepted")
	}
}
