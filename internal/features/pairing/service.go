// Package pairing implements couple-code generation, joining, and the
// wedding-date flow (docs/BACKEND_GO.md §8.1).
package pairing

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/sofolit/boureminder/api/internal/db/sqlc"
	"github.com/sofolit/boureminder/api/internal/httpx"
	"github.com/sofolit/boureminder/api/internal/jobs"
)

// codeAlphabet excludes 0/O/1/I, which are easy to confuse when read off
// a phone screen or typed by hand (docs/PRD.md §4.1).
const codeAlphabet = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

const codeTTL = 24 * time.Hour

type Service struct {
	db    *pgxpool.Pool
	q     *sqlc.Queries
	river *river.Client[pgx.Tx]
}

func NewService(db *pgxpool.Pool, riverClient *river.Client[pgx.Tx]) *Service {
	return &Service{db: db, q: sqlc.New(db), river: riverClient}
}

// CreateCode generates a fresh 6-character pair code for the caller,
// expiring any of their own unused codes first so only one is ever
// live at a time.
func (s *Service) CreateCode(ctx context.Context, ownerID uuid.UUID) (sqlc.PairCode, error) {
	if err := s.q.ExpireUnusedPairCodesForOwner(ctx, ownerID); err != nil {
		return sqlc.PairCode{}, fmt.Errorf("expire old pair codes: %w", err)
	}
	code, err := randomCode()
	if err != nil {
		return sqlc.PairCode{}, fmt.Errorf("generate pair code: %w", err)
	}
	pc, err := s.q.CreatePairCode(ctx, sqlc.CreatePairCodeParams{
		Code:      code,
		OwnerID:   ownerID,
		ExpiresAt: time.Now().Add(codeTTL),
	})
	if err != nil {
		return sqlc.PairCode{}, fmt.Errorf("create pair code: %w", err)
	}
	return pc, nil
}

// Join redeems a pair code, creating the couple and the system special
// dates (reciprocal birthdays) and deed vaults in one transaction, per
// docs/BACKEND_GO.md §8.1.
func (s *Service) Join(ctx context.Context, me uuid.UUID, rawCode string) (uuid.UUID, error) {
	code := strings.ToUpper(strings.TrimSpace(rawCode))

	var coupleID uuid.UUID
	err := withTx(ctx, s.db, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		row, err := q.LockPairCode(ctx, code)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return httpx.ErrCode(404, "pair_code_not_found")
		case err != nil:
			return fmt.Errorf("lock pair code: %w", err)
		case row.UsedAt.Valid:
			return httpx.ErrCode(409, "pair_code_used")
		case time.Now().After(row.ExpiresAt):
			return httpx.ErrCode(410, "pair_code_expired")
		case row.OwnerID == me:
			return httpx.ErrCode(400, "pair_code_own")
		}
		owner := row.OwnerID

		pairIDs := []uuid.UUID{me, owner}
		paired, err := q.CountPairedAmong(ctx, pairIDs)
		if err != nil {
			return fmt.Errorf("count paired: %w", err)
		}
		if paired > 0 {
			return httpx.ErrCode(409, "already_paired")
		}

		coupleID = uuid.Must(uuid.NewV7())
		if _, err := q.CreateCouple(ctx, coupleID); err != nil {
			return fmt.Errorf("create couple: %w", err)
		}
		if err := q.AssignCoupleToUsers(ctx, sqlc.AssignCoupleToUsersParams{
			CoupleID: pgUUID(coupleID),
			UserIds:  pairIDs,
		}); err != nil {
			return fmt.Errorf("assign couple: %w", err)
		}
		if err := q.MarkPairCodeUsed(ctx, code); err != nil {
			return fmt.Errorf("mark pair code used: %w", err)
		}

		meUser, err := q.GetUser(ctx, me)
		if err != nil {
			return fmt.Errorf("get me: %w", err)
		}
		ownerUser, err := q.GetUser(ctx, owner)
		if err != nil {
			return fmt.Errorf("get owner: %w", err)
		}
		if err := createSystemDates(ctx, q, coupleID, meUser, ownerUser); err != nil {
			return err
		}
		if err := createDeedVaults(ctx, q, coupleID, me, owner); err != nil {
			return err
		}

		_, err = s.river.InsertTx(ctx, tx, jobs.PushArgs{
			UserIDs:          pairIDs,
			NotificationKind: "paired",
			Route:            "/",
		}, nil)
		if err != nil {
			return fmt.Errorf("enqueue paired push: %w", err)
		}
		return nil
	})
	return coupleID, err
}

// createSystemDates adds each partner's birthday as a system special
// date, notifying only the OTHER partner so the birthday stays a
// surprise (docs/PRD.md §4.2). The anniversary date is created
// separately once the wedding date is set.
func createSystemDates(ctx context.Context, q *sqlc.Queries, coupleID uuid.UUID, a, b sqlc.User) error {
	for _, pair := range []struct {
		owner, notify sqlc.User
		kind          string
		title         string
	}{
		{a, b, birthdayKindFor(a), "জন্মদিন"},
		{b, a, birthdayKindFor(b), "জন্মদিন"},
	} {
		if pair.owner.BirthdayMonth == nil || pair.owner.BirthdayDay == nil {
			continue // profile setup should always set this, but never block pairing on it
		}
		_, err := q.CreateSpecialDate(ctx, sqlc.CreateSpecialDateParams{
			ID:            uuid.Must(uuid.NewV7()),
			CoupleID:      coupleID,
			Kind:          pair.kind,
			Title:         pair.title,
			Emoji:         "🎉",
			Month:         *pair.owner.BirthdayMonth,
			Day:           *pair.owner.BirthdayDay,
			Year:          pair.owner.BirthdayYear,
			RepeatsYearly: true,
			NotifyUserIds: []uuid.UUID{pair.notify.ID},
			IsSystem:      true,
			CreatedBy:     pair.owner.ID,
		})
		if err != nil {
			return fmt.Errorf("create system special date for %s: %w", pair.owner.ID, err)
		}
	}
	return nil
}

func birthdayKindFor(u sqlc.User) string {
	if u.Role != nil && *u.Role == "husband" {
		return "husband_birthday"
	}
	return "wife_birthday"
}

// createDeedVaults gives each partner their own vault, defaulting to the
// anniversary cycle (recomputed once the wedding date is known; falls
// back to 6 months until then, per docs/BACKEND_GO.md §8.2).
func createDeedVaults(ctx context.Context, q *sqlc.Queries, coupleID, a, b uuid.UUID) error {
	defaultUnlock := time.Now().AddDate(0, 6, 0)
	for _, pair := range [][2]uuid.UUID{{a, b}, {b, a}} {
		if _, err := q.CreateDeedVault(ctx, sqlc.CreateDeedVaultParams{
			AuthorID:    pair[0],
			CoupleID:    coupleID,
			RecipientID: pair[1],
			UnlockAt:    defaultUnlock,
		}); err != nil {
			return fmt.Errorf("create deed vault for %s: %w", pair[0], err)
		}
	}
	return nil
}

func randomCode() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, 6)
	for i, b := range buf {
		out[i] = codeAlphabet[int(b)%len(codeAlphabet)]
	}
	return string(out), nil
}

func withTx(ctx context.Context, pool *pgxpool.Pool, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // rollback after commit is a no-op

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

func pgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}
