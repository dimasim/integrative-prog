package service

import (
	"context"
	"errors"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/repository"
)

// ── Interface ─────────────────────────────────────────────────────────────────

type CommentService interface {
	Create(ctx context.Context, postID int64, req domain.CreateCommentRequest, callerID int64) (*domain.CommentResponse, error)
	ListByPost(ctx context.Context, postID int64) (*domain.CommentListResponse, error)
	Delete(ctx context.Context, commentID int64, callerID int64, callerRole string) error
}

// ── Implementation ────────────────────────────────────────────────────────────

type commentService struct {
	commentRepo repository.CommentRepository
	postRepo    repository.PostRepository // used to verify the parent post exists
}

func NewCommentService(commentRepo repository.CommentRepository, postRepo repository.PostRepository) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		postRepo:    postRepo,
	}
}

func (s *commentService) Create(ctx context.Context, postID int64, req domain.CreateCommentRequest, callerID int64) (*domain.CommentResponse, error) {
	// Verify parent post exists and is not soft-deleted.
	if _, err := s.postRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return nil, domain.ErrPostNotFound
		}
		return nil, domain.ErrInternalServer
	}

	comment := &domain.Comment{
		PostID: postID,
		UserID: callerID,
		Body:   req.Body,
	}

	created, err := s.commentRepo.Create(ctx, comment)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	resp := domain.ToCommentResponse(*created)
	return &resp, nil
}

func (s *commentService) ListByPost(ctx context.Context, postID int64) (*domain.CommentListResponse, error) {
	// Verify parent post exists.
	if _, err := s.postRepo.GetByID(ctx, postID); err != nil {
		if errors.Is(err, domain.ErrPostNotFound) {
			return nil, domain.ErrPostNotFound
		}
		return nil, domain.ErrInternalServer
	}

	comments, total, err := s.commentRepo.ListByPost(ctx, postID)
	if err != nil {
		return nil, domain.ErrInternalServer
	}

	resp := make([]domain.CommentResponse, 0, len(comments))
	for _, c := range comments {
		resp = append(resp, domain.ToCommentResponse(c))
	}

	return &domain.CommentListResponse{
		Comments: resp,
		Total:    total,
	}, nil
}

func (s *commentService) Delete(ctx context.Context, commentID int64, callerID int64, callerRole string) error {
	// Admin may delete any comment; others must own it.
	if callerRole != "admin" {
		ownerID, err := s.commentRepo.GetOwnerID(ctx, commentID)
		if err != nil {
			if errors.Is(err, domain.ErrCommentNotFound) {
				return domain.ErrCommentNotFound
			}
			return domain.ErrInternalServer
		}
		if ownerID != callerID {
			return domain.ErrForbidden
		}
	}

	if err := s.commentRepo.SoftDelete(ctx, commentID); err != nil {
		if errors.Is(err, domain.ErrCommentNotFound) {
			return domain.ErrCommentNotFound
		}
		return domain.ErrInternalServer
	}
	return nil
}
