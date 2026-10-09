package me

import (
	"github.com/google/uuid"

	"github.com/sofolit/boureminder/api/internal/db/sqlc"
)

// profile is the JSON shape returned by GET/PATCH /v1/me and the
// bootstrap endpoint. snake_case per docs/BACKEND_GO.md §4.
type profile struct {
	ID            uuid.UUID  `json:"id"`
	DisplayName   string     `json:"display_name"`
	Nickname      string     `json:"nickname"`
	Role          *string    `json:"role"`
	BirthdayMonth *int16     `json:"birthday_month"`
	BirthdayDay   *int16     `json:"birthday_day"`
	BirthdayYear  *int16     `json:"birthday_year"`
	PhotoKey      *string    `json:"photo_key"`
	CoupleID      *uuid.UUID `json:"couple_id"`
	Locale        string     `json:"locale"`
}

func toProfile(u sqlc.User) profile {
	p := profile{
		ID:            u.ID,
		DisplayName:   u.DisplayName,
		Nickname:      u.Nickname,
		Role:          u.Role,
		BirthdayMonth: u.BirthdayMonth,
		BirthdayDay:   u.BirthdayDay,
		BirthdayYear:  u.BirthdayYear,
		PhotoKey:      u.PhotoKey,
		Locale:        u.Locale,
	}
	if u.CoupleID.Valid {
		id := uuid.UUID(u.CoupleID.Bytes)
		p.CoupleID = &id
	}
	return p
}
