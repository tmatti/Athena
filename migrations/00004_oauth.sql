-- +goose Up
CREATE TABLE oauth_clients (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name          text NOT NULL DEFAULT '',
    redirect_uris text[] NOT NULL,
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth_auth_codes (
    code_hash      bytea PRIMARY KEY,
    client_id      uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    redirect_uri   text NOT NULL,
    code_challenge text NOT NULL,
    scope          text NOT NULL DEFAULT '',
    resource       text NOT NULL DEFAULT '',
    subject        text NOT NULL,
    expires_at     timestamptz NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE oauth_tokens (
    token_hash bytea PRIMARY KEY,
    kind       text NOT NULL CHECK (kind IN ('access', 'refresh')),
    client_id  uuid NOT NULL REFERENCES oauth_clients(id) ON DELETE CASCADE,
    subject    text NOT NULL,
    scope      text NOT NULL DEFAULT '',
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX oauth_tokens_expires_idx ON oauth_tokens (expires_at);
CREATE INDEX oauth_auth_codes_expires_idx ON oauth_auth_codes (expires_at);

-- +goose Down
DROP TABLE oauth_tokens;
DROP TABLE oauth_auth_codes;
DROP TABLE oauth_clients;
