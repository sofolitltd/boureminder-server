-- +goose Up
CREATE TABLE users (
    id               uuid PRIMARY KEY,
    firebase_uid     text NOT NULL UNIQUE,
    display_name     text NOT NULL DEFAULT '',
    nickname         text NOT NULL DEFAULT '',
    role             text CHECK (role IN ('wife','husband')),
    birthday_month   smallint CHECK (birthday_month BETWEEN 1 AND 12),
    birthday_day     smallint CHECK (birthday_day BETWEEN 1 AND 31),
    birthday_year    smallint,
    photo_key        text,
    couple_id        uuid,
    locale           text NOT NULL DEFAULT 'bn',
    settings         jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    deletion_requested_at timestamptz
);

CREATE TABLE couples (
    id                 uuid PRIMARY KEY,
    wedding_month      smallint,
    wedding_day        smallint,
    wedding_year       smallint,
    wedding_set_by     uuid REFERENCES users(id),
    wedding_confirmed  boolean NOT NULL DEFAULT false,
    change_seq         bigint NOT NULL DEFAULT 0,
    unpair_requested_at timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE users ADD CONSTRAINT users_couple_fk
    FOREIGN KEY (couple_id) REFERENCES couples(id) ON DELETE SET NULL;
CREATE INDEX users_couple_idx ON users(couple_id);

CREATE TABLE pair_codes (
    code        text PRIMARY KEY,
    owner_id    uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  timestamptz NOT NULL,
    used_at     timestamptz
);

CREATE TABLE devices (
    token        text PRIMARY KEY,
    user_id      uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform     text NOT NULL CHECK (platform IN ('android','ios')),
    app_version  text NOT NULL,
    locale       text NOT NULL DEFAULT 'bn',
    last_seen_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX devices_user_idx ON devices(user_id);

CREATE TABLE change_log (
    couple_id   uuid NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    seq         bigint NOT NULL,
    entity      text NOT NULL,
    entity_id   uuid NOT NULL,
    op          text NOT NULL CHECK (op IN ('upsert','delete')),
    visible_to  uuid,
    changed_at  timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (couple_id, seq)
);

CREATE TABLE processed_ops (
    op_id       uuid PRIMARY KEY,
    user_id     uuid NOT NULL,
    result      jsonb NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE uploads (
    key          text PRIMARY KEY,
    owner_id     uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    couple_id    uuid,
    purpose      text NOT NULL,
    content_type text NOT NULL,
    size_bytes   bigint NOT NULL,
    private_to   uuid,
    confirmed_at timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE uploads, processed_ops, change_log, devices, pair_codes;
ALTER TABLE users DROP CONSTRAINT users_couple_fk;
DROP TABLE couples, users;
