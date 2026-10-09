-- +goose Up
CREATE TABLE deed_vaults (
    author_id              uuid PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    couple_id              uuid NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    recipient_id           uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cycle_type             text NOT NULL DEFAULT 'anniversary' CHECK (cycle_type IN ('1m','6m','anniversary')),
    cycle_started_at       timestamptz NOT NULL DEFAULT now(),
    unlock_at              timestamptz NOT NULL,
    entry_count            int NOT NULL DEFAULT 0,
    streak                 int NOT NULL DEFAULT 0,
    last_entry_date        date,
    show_count_to_partner  boolean NOT NULL DEFAULT true,
    updated_at             timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX deed_vaults_unlock_idx ON deed_vaults(unlock_at);

-- +goose Down
DROP TABLE deed_vaults;
