package mock

import (
	"context"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/service"
)

var _ service.CommentService = (*MockCommentService)(nil)

// MockCommentService adalah no-op mock untuk CommentService.
// PostHandler membutuhkan CommentService di constructor-nya, tapi test
// untuk UpdatePost tidak akan pernah memanggil method comment — so we
// provide stubs that panic on unexpected calls.
type MockCommentService struct {
	CreateFn    func(ctx context.Context, postID int64, req domain.CreateCommentRequest, callerID int64) (*domain.CommentResponse, error)
	ListByPostFn func(ctx context.Context, postID int64) (*domain.CommentListResponse, error)
	DeleteFn    func(ctx context.Context, commentID int64, callerID int64, callerRole string) error
}

func (m *MockCommentService) Create(ctx context.Context, postID int64, req domain.CreateCommentRequest, callerID int64) (*domain.CommentResponse, error) {
	if m.CreateFn == nil {
		panic("MockCommentService.CreateFn not set — unexpected call in this test")
	}
	return m.CreateFn(ctx, postID, req, callerID)
}

func (m *MockCommentService) ListByPost(ctx context.Context, postID int64) (*domain.CommentListResponse, error) {
	if m.ListByPostFn == nil {
		panic("MockCommentService.ListByPostFn not set — unexpected call in this test")
	}
	return m.ListByPostFn(ctx, postID)
}

func (m *MockCommentService) Delete(ctx context.Context, commentID int64, callerID int64, callerRole string) error {
	if m.DeleteFn == nil {
		panic("MockCommentService.DeleteFn not set — unexpected call in this test")
	}
	return m.DeleteFn(ctx, commentID, callerID, callerRole)
}
