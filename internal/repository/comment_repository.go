package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dimasim/integrative-prog/internal/domain"
)

// ── Interface ─────────────────────────────────────────────────────────────────

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) (*domain.CommentWithAuthor, error)
	GetByID(ctx context.Context, id int64) (*domain.CommentWithAuthor, error)
	ListByPost(ctx context.Context, postID int64) ([]domain.CommentWithAuthor, int, error)
	SoftDelete(ctx context.Context, id int64) error
	// GetOwnerID is a lightweight helper used by the service for ownership checks.
	GetOwnerID(ctx context.Context, id int64) (int64, error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type commentRepository struct {
	db *sql.DB

	stmtCreate     *sql.Stmt
	stmtGetByID    *sql.Stmt
	stmtGetOwnerID *sql.Stmt
	stmtListByPost *sql.Stmt
	stmtCountByPost *sql.Stmt
	stmtSoftDelete *sql.Stmt
}

func NewCommentRepository(db *sql.DB) (CommentRepository, error) {
	r := &commentRepository{db: db}

	var err error

	r.stmtCreate, err = db.Prepare(`
		INSERT INTO comments (post_id, user_id, body)
		VALUES ($1, $2, $3)
		RETURNING id, post_id, user_id, body, created_at, updated_at, deleted_at`)
	if err != nil {
		return nil, err
	}

	r.stmtGetByID, err = db.Prepare(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, c.updated_at, c.deleted_at,
		       u.name AS author_name
		FROM   comments c
		JOIN   users    u ON u.id = c.user_id
		WHERE  c.id = $1
		  AND  c.deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtGetOwnerID, err = db.Prepare(`
		SELECT user_id FROM comments WHERE id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtListByPost, err = db.Prepare(`
		SELECT c.id, c.post_id, c.user_id, c.body, c.created_at, c.updated_at, c.deleted_at,
		       u.name AS author_name
		FROM   comments c
		JOIN   users    u ON u.id = c.user_id
		WHERE  c.post_id    = $1
		  AND  c.deleted_at IS NULL
		ORDER  BY c.created_at ASC`)
	if err != nil {
		return nil, err
	}

	r.stmtCountByPost, err = db.Prepare(`
		SELECT COUNT(*) FROM comments WHERE post_id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	r.stmtSoftDelete, err = db.Prepare(`
		UPDATE comments SET deleted_at = NOW()
		WHERE  id = $1 AND deleted_at IS NULL`)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (r *commentRepository) Create(ctx context.Context, comment *domain.Comment) (*domain.CommentWithAuthor, error) {
	var c domain.Comment
	err := r.stmtCreate.QueryRowContext(ctx, comment.PostID, comment.UserID, comment.Body).
		Scan(&c.ID, &c.PostID, &c.UserID, &c.Body, &c.CreatedAt, &c.UpdatedAt, &c.DeletedAt)
	if err != nil {
		return nil, err
	}
	return r.GetByID(ctx, c.ID)
}

func (r *commentRepository) GetByID(ctx context.Context, id int64) (*domain.CommentWithAuthor, error) {
	var c domain.CommentWithAuthor
	err := r.stmtGetByID.QueryRowContext(ctx, id).
		Scan(&c.ID, &c.PostID, &c.UserID, &c.Body,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
			&c.AuthorName)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrCommentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *commentRepository) GetOwnerID(ctx context.Context, id int64) (int64, error) {
	var ownerID int64
	err := r.stmtGetOwnerID.QueryRowContext(ctx, id).Scan(&ownerID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, domain.ErrCommentNotFound
	}
	return ownerID, err
}

func (r *commentRepository) ListByPost(ctx context.Context, postID int64) ([]domain.CommentWithAuthor, int, error) {
	var total int
	if err := r.stmtCountByPost.QueryRowContext(ctx, postID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := r.stmtListByPost.QueryContext(ctx, postID)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var comments []domain.CommentWithAuthor
	for rows.Next() {
		var c domain.CommentWithAuthor
		if err := rows.Scan(
			&c.ID, &c.PostID, &c.UserID, &c.Body,
			&c.CreatedAt, &c.UpdatedAt, &c.DeletedAt,
			&c.AuthorName,
		); err != nil {
			return nil, 0, err
		}
		comments = append(comments, c)
	}
	return comments, total, rows.Err()
}

func (r *commentRepository) SoftDelete(ctx context.Context, id int64) error {
	res, err := r.stmtSoftDelete.ExecContext(ctx, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrCommentNotFound
	}
	return nil
}
