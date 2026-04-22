package service

import (
	"context"
	"errors"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/repository"
)

// ── Interface ─────────────────────────────────────────────────────────────────

type PostService interface {
	Create(ctx context.Context, req domain.CreatePostRequest, callerID int64) (*domain.PostResponse, error)
	GetByID(ctx context.Context, id int64) (*domain.PostResponse, error)
	List(ctx context.Context, limit, offset int) (*domain.PostListResponse, error)
	Update(ctx context.Context, id int64, req domain.UpdatePostRequest, callerID int64, callerRole string) (*domain.PostResponse, error)
	Delete(ctx context.Context, id int64, callerID int64, callerRole string) error
}

// ── Implementation ────────────────────────────────────────────────────────────

type postService struct {
	repo repository.PostRepository
}

func NewPostService(repo repository.PostRepository) PostService {
	return &postService{repo: repo}
}

func (s *postService) Create(ctx context.Context, req domain.CreatePostRequest, callerID int64) (*domain.PostResponse, error) {
	post := &domain.Post{
		UserID: callerID,
		Title:  req.Title,
		Body:   req.Body,
	}

	created, err := s.repo.Create(ctx, post)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	resp := domain.ToPostResponse(*created)
	return &resp, nil
}

func (s *postService) GetByID(ctx context.Context, id int64) (*domain.PostResponse, error) {
	p, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return nil, domain.ErrPostNotFound
		}
		return nil, domain.ErrInternalServer
	}

	resp := domain.ToPostResponse(*p)
	return &resp, nil
}

func (s *postService) List(ctx context.Context, limit, offset int) (*domain.PostListResponse, error) {
	// Apply safe defaults so callers never pass zeros.
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	posts, total, err := s.repo.List(ctx, limit, offset)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	resp := make([]domain.PostResponse, 0, len(posts))
	for _, p := range posts {
		resp = append(resp, domain.ToPostResponse(p))
	}

	return &domain.PostListResponse{
		Posts:  resp,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *postService) Update(ctx context.Context, id int64, req domain.UpdatePostRequest, callerID int64, callerRole string) (*domain.PostResponse, error) {
	// Ownership check — only the owner may update (no admin bypass for UPDATE).
	if err := s.assertOwner(ctx, id, callerID); err != nil {
		return nil, err
	}

	// Build a partial Post; COALESCE in the query handles nil fields.
	patch := &domain.Post{ID: id}
	if req.Title != nil {
		patch.Title = *req.Title
	}
	if req.Body != nil {
		patch.Body = *req.Body
	}

	updated, err := s.repo.Update(ctx, patch)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return nil, domain.ErrPostNotFound
		}
		return nil, domain.ErrInternalServer
	}

	resp := domain.ToPostResponse(*updated)
	return &resp, nil
}

func (s *postService) Delete(ctx context.Context, id int64, callerID int64, callerRole string) error {
	// Admin may delete any post; others must own it.
	if callerRole != "admin" {
		if err := s.assertOwner(ctx, id, callerID); err != nil {
			return err
		}
	}

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return domain.ErrPostNotFound
		}
		return domain.ErrInternalServer
	}
	return nil
}

// assertOwner returns ErrForbidden when callerID is not the post owner.
// It propagates ErrPostNotFound if the post does not exist.
func (s *postService) assertOwner(ctx context.Context, postID, callerID int64) error {
	ownerID, err := s.repo.GetOwnerID(ctx, postID)
	if err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return domain.ErrPostNotFound
		}
		return domain.ErrInternalServer
	}
	if ownerID != callerID {
		return domain.ErrForbidden
	}
	return nil
}
