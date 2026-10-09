-- +goose Up
CREATE TABLE special_dates (
    id               uuid PRIMARY KEY,
    couple_id        uuid NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    kind             text NOT NULL CHECK (kind IN ('wife_birthday','husband_birthday','anniversary','custom')),
    title            text NOT NULL,
    emoji            text NOT NULL DEFAULT '🎉',
    month            smallint NOT NULL CHECK (month BETWEEN 1 AND 12),
    day              smallint NOT NULL CHECK (day BETWEEN 1 AND 31),
    year             smallint,
    repeats_yearly   boolean NOT NULL DEFAULT true,
    reminder_offsets smallint[] NOT NULL DEFAULT '{7,3,1,0}',
    reminder_time    time NOT NULL DEFAULT '08:00',
    notify_user_ids  uuid[] NOT NULL,
    is_system        boolean NOT NULL DEFAULT false,
    created_by       uuid NOT NULL REFERENCES users(id),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deleted_at       timestamptz
);
CREATE INDEX special_dates_couple_idx ON special_dates(couple_id) WHERE deleted_at IS NULL;

-- +goose Down
DROP TABLE special_dates;
