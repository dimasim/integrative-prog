CREATE TABLE IF NOT EXISTS comments (
    id          BIGSERIAL PRIMARY KEY,
    post_id     BIGINT  NOT NULL REFERENCES posts(id)  ON DELETE CASCADE,
    user_id     BIGINT  NOT NULL REFERENCES users(id)  ON DELETE CASCADE,
    body        TEXT    NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ
);

CREATE INDEX idx_comments_post_id    ON comments(post_id);
CREATE INDEX idx_comments_user_id    ON comments(user_id);
CREATE INDEX idx_comments_deleted_at ON comments(deleted_at) WHERE deleted_at IS NULL;
