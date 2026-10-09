// Package me implements account bootstrap, profile read/update, and FCM
// device registration (docs/BACKEND_GO.md §7 "me" rows).
package me

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	appauth "github.com/sofolit/boureminder/api/internal/auth"
	"github.com/sofolit/boureminder/api/internal/db/sqlc"
)

type Service struct {
	db *pgxpool.Pool
	q  *sqlc.Queries
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db, q: sqlc.New(db)}
}

// ActorByFirebaseUID implements auth.UserLookup.
func (s *Service) ActorByFirebaseUID(ctx context.Context, firebaseUID string) (appauth.Actor, error) {
	user, err := s.q.GetUserByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return appauth.Actor{}, appauth.ErrUserNotFound
		}
		return appauth.Actor{}, fmt.Errorf("get user by firebase uid: %w", err)
	}

	actor := appauth.Actor{UserID: user.ID, FirebaseUID: user.FirebaseUid}
	if user.CoupleID.Valid {
		actor.CoupleID = user.CoupleID.Bytes
		partner, err := s.q.GetPartner(ctx, user.ID)
		if err == nil {
			actor.PartnerID = partner.ID
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return appauth.Actor{}, fmt.Errorf("get partner: %w", err)
		}
	}
	return actor, nil
}

// Bootstrap creates the users row for a freshly-verified Firebase UID on
// first sign-in, or returns the existing row — idempotent, since the
// client may retry after a dropped connection (docs/BACKEND_GO.md §7).
func (s *Service) Bootstrap(ctx context.Context, firebaseUID string) (sqlc.User, error) {
	user, err := s.q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:          uuid.Must(uuid.NewV7()),
		FirebaseUid: firebaseUID,
	})
	if err != nil {
		return sqlc.User{}, fmt.Errorf("bootstrap user: %w", err)
	}
	return user, nil
}

type UpdateProfileInput struct {
	DisplayName   *string
	Nickname      *string
	Role          *string
	BirthdayMonth *int16
	BirthdayDay   *int16
	BirthdayYear  *int16
	PhotoKey      *string
}

func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, in UpdateProfileInput) (sqlc.User, error) {
	user, err := s.q.UpdateProfile(ctx, sqlc.UpdateProfileParams{
		ID:            userID,
		DisplayName:   in.DisplayName,
		Nickname:      in.Nickname,
		Role:          in.Role,
		BirthdayMonth: in.BirthdayMonth,
		BirthdayDay:   in.BirthdayDay,
		BirthdayYear:  in.BirthdayYear,
		PhotoKey:      in.PhotoKey,
	})
	if err != nil {
		return sqlc.User{}, fmt.Errorf("update profile %s: %w", userID, err)
	}
	return user, nil
}

func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, token, platform, appVersion, locale string) error {
	if err := s.q.UpsertDevice(ctx, sqlc.UpsertDeviceParams{
		Token:      token,
		UserID:     userID,
		Platform:   platform,
		AppVersion: appVersion,
		Locale:     locale,
	}); err != nil {
		return fmt.Errorf("register device: %w", err)
	}
	return nil
}

func (s *Service) UnregisterDevice(ctx context.Context, token string) error {
	if err := s.q.DeleteDevice(ctx, token); err != nil {
		return fmt.Errorf("unregister device: %w", err)
	}
	return nil
}
