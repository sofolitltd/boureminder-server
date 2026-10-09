package pairing

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/sofolit/boureminder/api/internal/db/sqlc"
)

// SetWeddingDate records the wedding date. The first partner to call
// this sets it (unconfirmed); the second partner's call with the same
// date confirms it and creates the shared anniversary special date,
// notifying both of them (docs/PRD.md §4.1, §4.2).
func (s *Service) SetWeddingDate(ctx context.Context, actorID, coupleID uuid.UUID, partnerID uuid.UUID, month, day int16, year *int16) (sqlc.Couple, error) {
	var couple sqlc.Couple
	err := withTx(ctx, s.db, func(tx pgx.Tx) error {
		q := s.q.WithTx(tx)

		existing, err := q.GetCouple(ctx, coupleID)
		if err != nil {
			return fmt.Errorf("get couple: %w", err)
		}

		if existing.WeddingSetBy.Valid {
			// Someone already proposed a date: this call either confirms
			// it (same date, other partner) or just updates it.
			sameDate := existing.WeddingMonth != nil && *existing.WeddingMonth == month &&
				existing.WeddingDay != nil && *existing.WeddingDay == day
			confirmedByOther := sameDate && uuid.UUID(existing.WeddingSetBy.Bytes) != actorID
			couple, err = q.SetWeddingDate(ctx, sqlc.SetWeddingDateParams{
				ID:               coupleID,
				WeddingMonth:     &month,
				WeddingDay:       &day,
				WeddingYear:      year,
				WeddingSetBy:     pgUUID(actorID),
				WeddingConfirmed: confirmedByOther || existing.WeddingConfirmed,
			})
			if err != nil {
				return fmt.Errorf("set wedding date: %w", err)
			}
			if confirmedByOther {
				if err := createAnniversaryDate(ctx, q, coupleID, actorID, partnerID, month, day, year); err != nil {
					return err
				}
			}
			return nil
		}

		couple, err = q.SetWeddingDate(ctx, sqlc.SetWeddingDateParams{
			ID:               coupleID,
			WeddingMonth:     &month,
			WeddingDay:       &day,
			WeddingYear:      year,
			WeddingSetBy:     pgUUID(actorID),
			WeddingConfirmed: false,
		})
		if err != nil {
			return fmt.Errorf("set wedding date: %w", err)
		}
		return nil
	})
	return couple, err
}

func createAnniversaryDate(ctx context.Context, q *sqlc.Queries, coupleID, a, b uuid.UUID, month, day int16, year *int16) error {
	_, err := q.CreateSpecialDate(ctx, sqlc.CreateSpecialDateParams{
		ID:            uuid.Must(uuid.NewV7()),
		CoupleID:      coupleID,
		Kind:          "anniversary",
		Title:         "বিবাহবার্ষিকী",
		Emoji:         "💍",
		Month:         month,
		Day:           day,
		Year:          year,
		RepeatsYearly: true,
		NotifyUserIds: []uuid.UUID{a, b},
		IsSystem:      true,
		CreatedBy:     a,
	})
	if err != nil {
		return fmt.Errorf("create anniversary special date: %w", err)
	}
	return nil
}
