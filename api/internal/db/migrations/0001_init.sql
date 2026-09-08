-- Migration 0001: initial schema for the MVP catalog database, per
-- KB/0010-mvp-scope.md's "Data model (Postgres)" section. Raw manifest JSON
-- is preserved verbatim per identity; fields that need querying or joining
-- (for example, computing verified/unverified/disputed split status) are
-- pulled into their own columns and tables alongside it.
--
-- Applied exactly once and tracked in schema_migrations (see db.go); later
-- schema changes are new numbered files in this directory, never edits to
-- this one, so the migration history stays a permanent record.

CREATE TABLE IF NOT EXISTS identities (
    url                  TEXT PRIMARY KEY,
    type                 TEXT NOT NULL CHECK (type IN ('artist', 'label')),
    name                 TEXT NOT NULL,
    public_key           TEXT NOT NULL,
    contact_email        TEXT NOT NULL,
    manifest_version     TEXT NOT NULL,
    raw_manifest         JSONB NOT NULL,
    fetched_at           TIMESTAMPTZ NOT NULL,
    ttl_seconds          INTEGER NOT NULL,
    history_head_hash    TEXT NOT NULL DEFAULT '',
    affiliated_label_url TEXT NOT NULL DEFAULT '',
    split_artist_pct     DOUBLE PRECISION,
    split_label_pct      DOUBLE PRECISION,
    last_error           TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS label_rosters (
    id               BIGSERIAL PRIMARY KEY,
    label_url        TEXT NOT NULL REFERENCES identities (url) ON DELETE CASCADE,
    artist_url       TEXT NOT NULL,
    split_artist_pct DOUBLE PRECISION NOT NULL,
    split_label_pct  DOUBLE PRECISION NOT NULL,
    UNIQUE (label_url, artist_url)
);

CREATE TABLE IF NOT EXISTS albums (
    id            BIGSERIAL PRIMARY KEY,
    identity_url  TEXT NOT NULL REFERENCES identities (url) ON DELETE CASCADE,
    album_id      TEXT NOT NULL,
    album_version INTEGER NOT NULL,
    album_name    TEXT NOT NULL,
    page_title    TEXT NOT NULL DEFAULT '',
    release_date  TEXT NOT NULL,
    images_front  TEXT NOT NULL,
    images_back   TEXT NOT NULL,
    images_insert JSONB NOT NULL DEFAULT '[]',
    download_zip  TEXT NOT NULL DEFAULT '',
    credits       JSONB,
    presentation  JSONB,
    UNIQUE (identity_url, album_id)
);

CREATE TABLE IF NOT EXISTS tracks (
    id       BIGSERIAL PRIMARY KEY,
    album_pk BIGINT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    track_id TEXT NOT NULL,
    side     TEXT NOT NULL DEFAULT '',
    number   TEXT NOT NULL,
    name     TEXT NOT NULL,
    duration TEXT NOT NULL DEFAULT '',
    file_url TEXT NOT NULL,
    lyrics   TEXT NOT NULL DEFAULT '',
    UNIQUE (album_pk, track_id)
);

CREATE TABLE IF NOT EXISTS album_splits (
    id           BIGSERIAL PRIMARY KEY,
    album_pk     BIGINT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    manifest_url TEXT NOT NULL,
    role         TEXT NOT NULL,
    percentage   DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS purchase_links (
    id       BIGSERIAL PRIMARY KEY,
    album_pk BIGINT NOT NULL REFERENCES albums (id) ON DELETE CASCADE,
    format   TEXT NOT NULL,
    url      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS contributions (
    id            BIGSERIAL PRIMARY KEY,
    identity_url  TEXT NOT NULL REFERENCES identities (url) ON DELETE CASCADE,
    manifest_url  TEXT NOT NULL,
    album_id      TEXT NOT NULL,
    album_version INTEGER NOT NULL,
    role          TEXT NOT NULL,
    percentage    DOUBLE PRECISION NOT NULL
);

CREATE TABLE IF NOT EXISTS merch_links (
    id           BIGSERIAL PRIMARY KEY,
    identity_url TEXT NOT NULL REFERENCES identities (url) ON DELETE CASCADE,
    label        TEXT NOT NULL,
    url          TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS history_entries (
    id              BIGSERIAL PRIMARY KEY,
    identity_url    TEXT NOT NULL REFERENCES identities (url) ON DELETE CASCADE,
    sequence_number INTEGER NOT NULL,
    timestamp       TEXT NOT NULL,
    type            TEXT NOT NULL,
    data            JSONB NOT NULL,
    previous_hash   TEXT NOT NULL DEFAULT '',
    signature       JSONB NOT NULL,
    UNIQUE (identity_url, sequence_number)
);
