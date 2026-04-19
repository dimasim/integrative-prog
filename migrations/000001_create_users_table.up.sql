CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    email      VARCHAR(255) NOT NULL,
    password   VARCHAR(255) NOT NULL,
    role       VARCHAR(20)  NOT NULL DEFAULT 'user'
                            CHECK (role IN ('admin', 'user')),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ  NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email
    ON users (email) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);
