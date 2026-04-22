package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dimasim/integrative-prog/internal/domain"
)

// ── Interface ─────────────────────────────────────────────────────────────────

type PostRepository interface {
	Create(ctx context.Context, post *domain.Post) (*domain.PostWithAuthor, error)
	GetByID(ctx context.Context, id int64) (*domain.PostWithAuthor, error)
	List(ctx context.Context, limit, offset int) ([]domain.PostWithAuthor, int, error)
	Update(ctx context.Context, post *domain.Post) (*domain.PostWithAuthor, error)
	SoftDelete(ctx context.Context, id int64) error
	// GetOwnerID is a lightweight helper used by the service for ownership checks.
	GetOwnerID(ctx context.Context, id int64) (int64, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type postRepository struct {
	db *sql.DB

	stmtCreate     *sql.Stmt
	stmtGetByID    *sql.Stmt
	stmtGetOwnerID *sql.Stmt
	stmtUpdate     *sql.Stmt
	stmtSoftDelete *sql.Stmt
	stmtCount      *sql.Stmt
}

func NewPostRepository(db *sql.DB) (PostRepository, error) {
	r := &postRepository{db: db}

	var err error

	r.stmtCreate, err = db.Prepare(`
		INSERT INTO posts (user_id, title, body)
		VALUES ($1, $2, $3)
		RETURNING id, user_id, title, body, created_at, updated_at, deleted_at`)
	if err != nil {
		return nil, err
	}

	// JOIN users to return author_name in one round-trip.
	r.stmtGetByID, err = db.Prepare(`
		SELECT p.id, p.user_id, p.title, p.body, p.created_at, p.updated_at, p.deleted_at,
		       u.name AS author_name
		FROM   posts p
		JOIN   users u ON u.id = p.user_id
		WHERE  p.id = $1
		  AND  p.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtGetOwnerID, err = db.Prepare(`
		SELECT user_id FROM posts WHERE id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtUpdate, err = db.Prepare(`
		UPDATE posts
		SET    title      = COALESCE($1, title),
		       body       = COALESCE($2, body),
		       updated_at = NOW()
		WHERE  id         = $3
		  AND  deleted_at IS NULL
		RETURNING id, user_id, title, body, created_at, updated_at, deleted_at`)
	if err != nil {
		return nil, err
	}

	r.stmtSoftDelete, err = db.Prepare(`
		UPDATE posts SET deleted_at = NOW()
		WHERE  id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtCount, err = db.Prepare(`
		SELECT COUNT(*) FROM posts WHERE deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *postRepository) Create(ctx context.Context, post *domain.Post) (*domain.PostWithAuthor, error) {
	// Insert the row, then fetch with JOIN for author_name.
	var p domain.Post
	err := r.stmtCreate.QueryRowContext(ctx, post.UserID, post.Title, post.Body).
		Scan(&p.ID, &p.UserID, &p.Title, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if err != nil {
		return nil, err
	}

	// Re-use GetByID to get the author_name via JOIN.
	return r.GetByID(ctx, p.ID)
}

func (r *postRepository) GetByID(ctx context.Context, id int64) (*domain.PostWithAuthor, error) {
	row := r.stmtGetByID.QueryRowContext(ctx, id)

	var p domain.PostWithAuthor
	err := row.Scan(
		&p.ID, &p.UserID, &p.Title, &p.Body,
		&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
		&p.AuthorName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *postRepository) GetOwnerID(ctx context.Context, id int64) (int64, error) {
	var ownerID int64
	err := r.stmtGetOwnerID.QueryRowContext(ctx, id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.ErrPostNotFound
	}
	return ownerID, err
}

func (r *postRepository) List(ctx context.Context, limit, offset int) ([]domain.PostWithAuthor, int, error) {
	// Count total (non-deleted) posts.
	var total int
	if err := r.stmtCount.QueryRowContext(ctx).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Parameterized list query — limit/offset are safe integer params.
	rows, err := r.db.QueryContext(ctx, `
		SELECT p.id, p.user_id, p.title, p.body, p.created_at, p.updated_at, p.deleted_at,
		       u.name AS author_name
		FROM   posts p
		JOIN   users u ON u.id = p.user_id
		WHERE  p.deleted_at IS NULL
		ORDER  BY p.created_at DESC
		LIMIT  $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var posts []domain.PostWithAuthor
	for rows.Next() {
		var p domain.PostWithAuthor
		if err := rows.Scan(
			&p.ID, &p.UserID, &p.Title, &p.Body,
			&p.CreatedAt, &p.UpdatedAt, &p.DeletedAt,
			&p.AuthorName,
		); err != nil {
			return nil, 0, err
		}
		posts = append(posts, p)
	}
	return posts, total, rows.Err()
}

func (r *postRepository) Update(ctx context.Context, post *domain.Post) (*domain.PostWithAuthor, error) {
	var p domain.Post
	err := r.stmtUpdate.QueryRowContext(ctx, post.Title, post.Body, post.ID).
		Scan(&p.ID, &p.UserID, &p.Title, &p.Body, &p.CreatedAt, &p.UpdatedAt, &p.DeletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrPostNotFound
	}
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, p.ID)
}

func (r *postRepository) SoftDelete(ctx context.Context, id int64) error {
	res, err := r.stmtSoftDelete.ExecContext(ctx, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrPostNotFound
	}
	return nil
}
