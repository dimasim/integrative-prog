// Package mock menyediakan implementasi in-memory dari semua service interface.
// Mock ini digunakan khusus untuk unit/functional testing di layer handler.
// Tidak ada dependency ke database, JWT library, atau external service.
package mock

import (
	"context"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/service"
)

// Pastikan MockPostService selalu mengimplementasikan interface PostService.
// Baris ini akan gagal kompilasi jika ada method yang kurang — early detection.
var _ service.PostService = (*MockPostService)(nil)

// ── Call recorder ─────────────────────────────────────────────────────────────
// Setiap call ke mock direkam ke slice agar test bisa mem-verifikasi
// bahwa handler benar-benar meneruskan argumen yang tepat ke service.

// UpdateCall merekam argumen yang diterima saat Update() dipanggil.
type UpdateCall struct {
	ID         int64
	Req        domain.UpdatePostRequest
	CallerID   int64
	CallerRole string
}

// ── MockPostService ───────────────────────────────────────────────────────────

// MockPostService adalah implementasi test-double dari service.PostService.
// Setiap method dapat di-override dengan mengisi field Fn-nya sebelum test dijalankan.
// Jika field Fn tidak diisi, method akan panic dengan pesan yang jelas — ini
// mencegah test melewati mock call yang seharusnya tidak terjadi.
type MockPostService struct {
	// Fn — injectable behavior per method
	CreateFn  func(ctx context.Context, req domain.CreatePostRequest, callerID int64) (*domain.PostResponse, error)
	GetByIDFn func(ctx context.Context, id int64) (*domain.PostResponse, error)
	ListFn    func(ctx context.Context, limit, offset int) (*domain.PostListResponse, error)
	UpdateFn  func(ctx context.Context, id int64, req domain.UpdatePostRequest, callerID int64, callerRole string) (*domain.PostResponse, error)
	DeleteFn  func(ctx context.Context, id int64, callerID int64, callerRole string) error

	// Recorder — untuk verifikasi call arguments
	UpdateCalls []UpdateCall
}

func (m *MockPostService) Create(ctx context.Context, req domain.CreatePostRequest, callerID int64) (*domain.PostResponse, error) {
	if m.CreateFn == nil {
		panic("MockPostService.CreateFn not set")
	}
	return m.CreateFn(ctx, req, callerID)
}

func (m *MockPostService) GetByID(ctx context.Context, id int64) (*domain.PostResponse, error) {
	if m.GetByIDFn == nil {
		panic("MockPostService.GetByIDFn not set")
	}
	return m.GetByIDFn(ctx, id)
}

func (m *MockPostService) List(ctx context.Context, limit, offset int) (*domain.PostListResponse, error) {
	if m.ListFn == nil {
		panic("MockPostService.ListFn not set")
	}
	return m.ListFn(ctx, limit, offset)
}

func (m *MockPostService) Update(ctx context.Context, id int64, req domain.UpdatePostRequest, callerID int64, callerRole string) (*domain.PostResponse, error) {
	if m.UpdateFn == nil {
		panic("MockPostService.UpdateFn not set")
	}
	// Rekam call sebelum delegate ke Fn
	m.UpdateCalls = append(m.UpdateCalls, UpdateCall{
		ID:         id,
		Req:        req,
		CallerID:   callerID,
		CallerRole: callerRole,
	})
	return m.UpdateFn(ctx, id, req, callerID, callerRole)
}

func (m *MockPostService) Delete(ctx context.Context, id int64, callerID int64, callerRole string) error {
	if m.DeleteFn == nil {
		panic("MockPostService.DeleteFn not set")
	}
	return m.DeleteFn(ctx, id, callerID, callerRole)
}
