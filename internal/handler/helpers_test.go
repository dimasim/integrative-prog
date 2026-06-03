// Package handler_test berisi functional test untuk semua Gin handler.
// File ini (helpers_test.go) menyediakan fungsi-fungsi utilitas yang di-share
// oleh semua file _test.go di package ini.
//
// Konvensi penamaan:
//   - File test: <handler_name>_test.go
//   - Helper:    helpers_test.go (file ini)
//   - Mock:      internal/mock/<service_name>_mock.go
package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/handler"
	"github.com/dimasim/integrative-prog/internal/middleware"
	"github.com/dimasim/integrative-prog/internal/mock"
	"github.com/dimasim/integrative-prog/internal/service"
)

// ── Router factory ────────────────────────────────────────────────────────────

// routerConfig membungkus semua dependency yang dibutuhkan untuk membangun
// test router. Field yang nil akan diganti dengan no-op mock secara otomatis.
type routerConfig struct {
	postSvc    service.PostService
	commentSvc service.CommentService
}

// newTestRouter membangun Gin engine dalam mode TestMode dengan:
//   - PostHandler yang di-wire ke mock services
//   - jwtAuth diganti dengan fakeAuthMiddleware agar test tidak perlu token JWT sungguhan
//
// fakeAuth meng-inject callerID dan callerRole langsung ke Gin context,
// persis seperti yang dilakukan middleware.JWTAuth() di production.
func newTestRouter(cfg routerConfig) *gin.Engine {
	gin.SetMode(gin.TestMode)
	if cfg.commentSvc == nil {
		cfg.commentSvc = &mock.MockCommentService{} // no-op, tidak akan dipanggil
	}

	h := handler.NewPostHandler(cfg.postSvc, cfg.commentSvc)

	r := gin.New() // tanpa Logger/Recovery agar output test bersih
	v1 := r.Group("/api/v1")

	// Ganti JWTAuth dengan middleware sederhana yang membaca header custom.
	// Ini mengisolasi test handler dari JWT library — kita tidak perlu
	// generate token sungguhan hanya untuk test handler.
	h.RegisterRoutes(v1, fakeAuthMiddleware())

	return r
}

// fakeAuthMiddleware membaca dua header test-only:
//   - X-Test-User-ID   → dikonversi ke int64, diset ke ContextKeyUserID
//   - X-Test-User-Role → diset ke ContextKeyRole
//
// Jika X-Test-User-ID kosong atau tidak ada, middleware abort 401 —
// perilaku ini konsisten dengan JWTAuth() di production.
func fakeAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userIDStr := c.GetHeader("X-Test-User-ID")
		if userIDStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, domain.APIError{
				Code:    http.StatusUnauthorized,
				Message: "unauthorized",
			})
			return
		}

		var userID int64
		fmt.Sscanf(userIDStr, "%d", &userID)

		role := c.GetHeader("X-Test-User-Role")
		if role == "" {
			role = "user"
		}

		// Key yang sama persis dengan yang dibaca handler:
		// c.GetInt64(string(middleware.ContextKeyUserID))
		c.Set(string(middleware.ContextKeyUserID), userID)
		c.Set(string(middleware.ContextKeyRole), role)
		c.Next()
	}
}

// ── Request factory ───────────────────────────────────────────────────────────

// requestOption adalah functional option untuk mengkustomisasi test request.
type requestOption func(*http.Request)

// withAuth menambahkan header auth fake ke request.
// userID adalah ID user yang diklaim sebagai caller.
// role adalah role user (default "user" jika kosong).
func withAuth(userID int64, role string) requestOption {
	return func(r *http.Request) {
		r.Header.Set("X-Test-User-ID", fmt.Sprintf("%d", userID))
		if role != "" {
			r.Header.Set("X-Test-User-Role", role)
		}
	}
}

// newJSONRequest membangun *http.Request dengan body JSON dan Content-Type yang benar.
// body boleh nil jika request tidak membutuhkan body (e.g., GET, DELETE).
func newJSONRequest(method, path string, body any, opts ...requestOption) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			panic("newJSONRequest: failed to encode body: " + err.Error())
		}
	}

	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")

	for _, opt := range opts {
		opt(req)
	}
	return req
}

// executeRequest menjalankan request ke router dan mengembalikan ResponseRecorder.
func executeRequest(router *gin.Engine, req *http.Request) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── Response decoder ──────────────────────────────────────────────────────────

// decodeAPIResponse men-decode JSON response ke domain.APIResponse.
// Field Data di-decode sebagai json.RawMessage agar test bisa
// men-decode lebih lanjut ke tipe yang spesifik jika dibutuhkan.
func decodeAPIResponse(w *httptest.ResponseRecorder) (domain.APIResponse, error) {
	var resp domain.APIResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	return resp, err
}

// decodeAPIError men-decode JSON response ke domain.APIError.
func decodeAPIError(w *httptest.ResponseRecorder) (domain.APIError, error) {
	var apiErr domain.APIError
	err := json.NewDecoder(w.Body).Decode(&apiErr)
	return apiErr, err
}

// ── Pointer helpers ───────────────────────────────────────────────────────────

// strPtr mengembalikan pointer ke string — berguna untuk mengisi UpdatePostRequest
// yang menggunakan *string untuk membedakan "tidak diisi" vs "diisi kosong".
func strPtr(s string) *string { return &s }
