package auth

import (
	"context"
	"fmt"

	firebase "firebase.google.com/go/v4"
	fbauth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"

	"github.com/sofolit/boureminder/api/internal/config"
)

// NewFirebase initializes the Firebase Admin SDK, scoped to Auth only
// (docs/SYSTEM_DESIGN.md: Firebase is never used for Firestore, Storage,
// or Functions). When FIREBASE_AUTH_EMULATOR_HOST is set, the Admin SDK
// automatically accepts the emulator's unsigned tokens instead of
// verifying against Google's public keys.
func NewFirebase(ctx context.Context, c config.Config) (*fbauth.Client, error) {
	var opts []option.ClientOption
	if c.FirebaseCreds != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(c.FirebaseCreds)))
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: c.FirebaseProject}, opts...)
	if err != nil {
		return nil, fmt.Errorf("init firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("init firebase auth client: %w", err)
	}
	return client, nil
}
