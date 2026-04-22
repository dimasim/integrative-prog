package domain

import (
	"errors"
	"time"
)

// ── Sentinel errors (extend existing set) ────────────────────────────────────

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
)

// ── Entities ─────────────────────────────────────────────────────────────────

// Post is the core database-mapped entity.
type Post struct {
	ID        int64      `db:"id"`
	UserID    int64      `db:"user_id"`
	Title     string     `db:"title"`
	Body      string     `db:"body"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// PostWithAuthor enriches Post with a JOIN-derived author name.
type PostWithAuthor struct {
	Post
	AuthorName string `db:"author_name"`
}

// Comment is the core database-mapped entity.
type Comment struct {
	ID        int64      `db:"id"`
	PostID    int64      `db:"post_id"`
	UserID    int64      `db:"user_id"`
	Body      string     `db:"body"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"`
}

// CommentWithAuthor enriches Comment with a JOIN-derived author name.
type CommentWithAuthor struct {
	Comment
	AuthorName string `db:"author_name"`
}

// ── DTOs ──────────────────────────────────────────────────────────────────────

// CreatePostRequest is the JSON body for POST /api/v1/posts.
// user_id is intentionally omitted — it is sourced from the JWT context.
type CreatePostRequest struct {
	Title string `json:"title" validate:"required,min=3,max=255"`
	Body  string `json:"body"  validate:"required,min=1"`
}

// UpdatePostRequest is the JSON body for PATCH /api/v1/posts/:id.
// Both fields are optional; at least one must be present (enforced in service).
type UpdatePostRequest struct {
	Title *string `json:"title" validate:"omitempty,min=3,max=255"`
	Body  *string `json:"body"  validate:"omitempty,min=1"`
}

// PostResponse is what callers receive back for a single post.
type PostResponse struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	AuthorName string    `json:"author_name"`
	Title      string    `json:"title"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// PostListRequest holds query-string pagination params.
type PostListRequest struct {
	Limit  int `form:"limit"  validate:"min=1,max=100"`
	Offset int `form:"offset" validate:"min=0"`
}

// PostListResponse wraps a paginated slice of posts.
type PostListResponse struct {
	Posts  []PostResponse `json:"posts"`
	Total  int            `json:"total"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
}

// CreateCommentRequest is the JSON body for POST /api/v1/posts/:post_id/comments.
type CreateCommentRequest struct {
	Body string `json:"body" validate:"required,min=1"`
}

// CommentResponse is what callers receive back for a single comment.
type CommentResponse struct {
	ID         int64     `json:"id"`
	PostID     int64     `json:"post_id"`
	UserID     int64     `json:"user_id"`
	AuthorName string    `json:"author_name"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// CommentListResponse wraps all comments for a post.
type CommentListResponse struct {
	Comments []CommentResponse `json:"comments"`
	Total    int               `json:"total"`
}

// ── Mapping helpers ───────────────────────────────────────────────────────────

func ToPostResponse(p PostWithAuthor) PostResponse {
	return PostResponse{
		ID:         p.ID,
		UserID:     p.UserID,
		AuthorName: p.AuthorName,
		Title:      p.Title,
		Body:       p.Body,
		CreatedAt:  p.CreatedAt,
		UpdatedAt:  p.UpdatedAt,
	}
}

func ToCommentResponse(c CommentWithAuthor) CommentResponse {
	return CommentResponse{
		ID:         c.ID,
		PostID:     c.PostID,
		UserID:     c.UserID,
		AuthorName: c.AuthorName,
		Body:       c.Body,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}
