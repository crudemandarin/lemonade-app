package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// Firebase verifies Firebase ID tokens with the Admin SDK: signature (against
// Google's cached public keys), issuer, audience, and expiry. It makes no per-request
// network call and needs no credentials file. When FIREBASE_AUTH_EMULATOR_HOST is set
// the SDK accepts the emulator's unsigned tokens.
type Firebase struct {
	client *fbauth.Client
}

// NewFirebase builds a verifier for one Firebase project.
func NewFirebase(ctx context.Context, projectID string) (*Firebase, error) {
	// Verification needs no service-account credentials, so none are looked up.
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, option.WithoutAuthentication())
	if err != nil {
		return nil, fmt.Errorf("firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth client: %w", err)
	}
	return &Firebase{client: client}, nil
}

func (f *Firebase) Verify(ctx context.Context, idToken string) (Identity, error) {
	// Deliberately not VerifyIDTokenAndCheckRevoked: that adds a network call to every
	// request, and a token lives an hour at most.
	tok, err := f.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return Identity{}, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	id := Identity{UID: tok.UID, Provider: tok.Firebase.SignInProvider}
	id.Email, _ = tok.Claims["email"].(string)
	id.EmailVerified, _ = tok.Claims["email_verified"].(bool)
	return id, nil
}
