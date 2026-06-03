package handler_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/mock"
)

// ── Fixtures ──────────────────────────────────────────────────────────────────
// Konstanta fixture dipusatkan di sini agar mudah diubah tanpa menyentuh
// logic test. Jika satu nilai berubah, hanya satu tempat yang perlu di-edit.

const (
	ownerUserID    int64 = 42
	nonOwnerUserID int64 = 99
	existingPostID int64 = 7
)

// fixedNow dipakai sebagai timestamp agar assertion pada field time tidak flaky.
var fixedNow = time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

// updatedPostResponse adalah response yang dikembalikan mock ketika update berhasil.
// Dibuat sebagai variabel package-level agar bisa dirujuk di assertions.
var updatedPostResponse = &domain.PostResponse{
	ID:         existingPostID,
	UserID:     ownerUserID,
	AuthorName: "Test Owner",
	Title:      "Judul Setelah Update",
	Body:       "Body lama tidak berubah",
	CreatedAt:  fixedNow,
	UpdatedAt:  fixedNow.Add(5 * time.Minute),
}

// ── Table-driven test: PATCH /api/v1/posts/:id ────────────────────────────────
//
// Mengapa table-driven?
//   - Semua skenario endpoint yang sama dikumpulkan dalam satu fungsi → mudah di-maintain
//   - Menambah skenario baru = menambah satu entry di slice, tidak perlu fungsi baru
//   - Output `go test -v` menampilkan nama sub-test dengan jelas (t.Run)
//
// Struktur setiap test case:
//   name        → label yang muncul di output `go test -v`
//   setupMock   → fungsi yang mengkonfigurasi behavior mock untuk skenario ini
//   buildReq    → fungsi yang membangun *http.Request (method, path, body, headers)
//   wantStatus  → HTTP status code yang diharapkan
//   assertBody  → fungsi opsional untuk verifikasi tambahan pada response body
//   assertMock  → fungsi opsional untuk verifikasi call ke mock (argumen, jumlah call, dll)

func TestUpdatePost(t *testing.T) {
	t.Parallel() // Semua sub-test bisa jalan paralel — masing-masing punya mock sendiri

	type testCase struct {
		name       string
		setupMock  func() *mock.MockPostService
		buildReq   func() *http.Request
		wantStatus int
		// UBAH BARIS INI:
		assertBody func(t *testing.T, body string) 
		assertMock func(t *testing.T, m *mock.MockPostService)
	}

	// Request body yang valid — dipakai ulang di beberapa test case
	validBody := domain.UpdatePostRequest{
		Title: strPtr("Judul Setelah Update"),
	}

	cases := []testCase{
		// ── HAPPY PATH ────────────────────────────────────────────────────────
		{
			name: "200 — owner berhasil update title post miliknya",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					UpdateFn: func(_ context.Context, id int64, req domain.UpdatePostRequest, callerID int64, callerRole string) (*domain.PostResponse, error) {
						// Service memvalidasi ownership, jika lolos kembalikan data terupdate
						return updatedPostResponse, nil
					},
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					validBody,
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusOK,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				require.Len(t, m.UpdateCalls, 1, "Update() harus dipanggil tepat satu kali")

				call := m.UpdateCalls[0]
				assert.Equal(t, existingPostID, call.ID, "post ID harus diteruskan dari URL param")
				assert.Equal(t, ownerUserID, call.CallerID, "callerID harus diambil dari JWT context, bukan body")
				assert.Equal(t, "user", call.CallerRole, "callerRole harus diteruskan ke service")
				assert.Equal(t, "Judul Setelah Update", *call.Req.Title)
				assert.Nil(t, call.Req.Body, "field Body tidak dikirim, harus nil (bukan string kosong)")
			},
			assertBody: func(t *testing.T, _ string) {
				// Body assertion dilakukan via wantStatus — cukup untuk happy path sederhana.
				// Jika perlu verifikasi field spesifik, decode di sini.
			},
		},

		{
			name: "200 — owner update semua field sekaligus (title + body)",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
						resp := *updatedPostResponse
						resp.Body = "Body Baru Setelah Update"
						return &resp, nil
					},
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					domain.UpdatePostRequest{
						Title: strPtr("Judul Baru"),
						Body:  strPtr("Body Baru Setelah Update"),
					},
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusOK,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				require.Len(t, m.UpdateCalls, 1)
				call := m.UpdateCalls[0]
				assert.NotNil(t, call.Req.Title)
				assert.NotNil(t, call.Req.Body)
			},
		},

		// ── FORBIDDEN ─────────────────────────────────────────────────────────
		{
			name: "403 — bukan owner mencoba update post orang lain",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
						// Service layer melakukan ownership check dan mengembalikan ErrForbidden
						// ketika callerID != ownerID. Handler harus memetakannya ke HTTP 403.
						return nil, domain.ErrForbidden
					},
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					validBody,
					withAuth(nonOwnerUserID, "user"), // user BUKAN owner
				)
			},
			wantStatus: http.StatusForbidden,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				require.Len(t, m.UpdateCalls, 1, "handler harus tetap meneruskan call ke service — ownership check ada di service, bukan handler")
				assert.Equal(t, nonOwnerUserID, m.UpdateCalls[0].CallerID)
			},
		},

		// ── NOT FOUND ─────────────────────────────────────────────────────────
		{
			name: "404 — post tidak ditemukan (atau sudah soft-deleted)",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
						return nil, domain.ErrPostNotFound
					},
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					validBody,
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusNotFound,
		},

		// ── INTERNAL SERVER ERROR ──────────────────────────────────────────────
		{
			name: "500 — service mengembalikan error internal (misal: DB down)",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
						return nil, domain.ErrInternalServer
					},
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					validBody,
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusInternalServerError,
		},

		// ── UNAUTHORIZED ──────────────────────────────────────────────────────
		{
			name: "401 — request tanpa JWT token sama sekali",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					// UpdateFn sengaja tidak di-set — jika handler salah dan tetap
					// meneruskan ke service tanpa auth, mock akan panic dan test gagal.
					// Ini membuktikan middleware abort sebelum handler dieksekusi.
				}
			},
			buildReq: func() *http.Request {
				// Tidak ada withAuth() → tidak ada header X-Test-User-ID
				return newJSONRequest(http.MethodPatch, "/api/v1/posts/7", validBody)
			},
			wantStatus: http.StatusUnauthorized,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				assert.Empty(t, m.UpdateCalls, "Update() tidak boleh dipanggil sama sekali jika request tidak terautentikasi")
			},
		},

		// ── VALIDATION ERROR ──────────────────────────────────────────────────
		{
			name: "400 — title terlalu pendek (min 3 karakter)",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{
					// UpdateFn tidak di-set — validasi harus gagal sebelum service dipanggil
				}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/7",
					domain.UpdatePostRequest{Title: strPtr("ab")}, // min=3, "ab" hanya 2 karakter
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusBadRequest,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				assert.Empty(t, m.UpdateCalls, "Update() tidak boleh dipanggil jika validasi gagal")
			},
		},

		{
			name: "400 — URL param :id bukan angka",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{}
			},
			buildReq: func() *http.Request {
				return newJSONRequest(
					http.MethodPatch,
					"/api/v1/posts/bukan-angka", // invalid ID
					validBody,
					withAuth(ownerUserID, "user"),
				)
			},
			wantStatus: http.StatusBadRequest,
			assertMock: func(t *testing.T, m *mock.MockPostService) {
				t.Helper()
				assert.Empty(t, m.UpdateCalls, "Update() tidak boleh dipanggil jika ID tidak valid")
			},
		},

		{
			name: "400 — request body bukan JSON valid",
			setupMock: func() *mock.MockPostService {
				return &mock.MockPostService{}
			},
			buildReq: func() *http.Request {
				// Kirim raw string yang bukan JSON
				req := newJSONRequest(http.MethodPatch, "/api/v1/posts/7", nil, withAuth(ownerUserID, "user"))
				// Override body dengan konten invalid
				req.Body = http.NoBody
				return req
			},
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tc := range cases {
		tc := tc // capture loop variable untuk t.Parallel()
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// ── Arrange ───────────────────────────────────────────────────────
			svcMock := tc.setupMock()
			router := newTestRouter(routerConfig{postSvc: svcMock})

			// ── Act ───────────────────────────────────────────────────────────
			req := tc.buildReq()
			w := executeRequest(router, req)

			// ── Assert: HTTP status ───────────────────────────────────────────
			assert.Equal(t, tc.wantStatus, w.Code,
				"status mismatch untuk skenario %q\nresponse body: %s",
				tc.name, w.Body.String(),
			)

			// ── Assert: response body ─────────────────────────────────────────
			if tc.assertBody != nil {
				// UBAH BARIS INI:
				tc.assertBody(t, w.Body.String())
			}

			// ── Assert: mock calls ────────────────────────────────────────────
			if tc.assertMock != nil {
				tc.assertMock(t, svcMock)
			}
		})
	}
}

// ── Test respons body yang spesifik ───────────────────────────────────────────
// Sub-test ini memverifikasi detail struktur JSON response — dipisah dari
// TestUpdatePost agar table-driven test di atas tetap ringkas.

func TestUpdatePost_ResponseBody(t *testing.T) {
	t.Parallel()

	svcMock := &mock.MockPostService{
		UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
			return updatedPostResponse, nil
		},
	}

	router := newTestRouter(routerConfig{postSvc: svcMock})
	req := newJSONRequest(
		http.MethodPatch,
		"/api/v1/posts/7",
		domain.UpdatePostRequest{Title: strPtr("Judul Setelah Update")},
		withAuth(ownerUserID, "user"),
	)

	w := executeRequest(router, req)

	require.Equal(t, http.StatusOK, w.Code)

	// Decode response envelope
	resp, err := decodeAPIResponse(w)
	require.NoError(t, err, "response body harus JSON valid")

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "post updated", resp.Message)
	assert.NotNil(t, resp.Data, "field data tidak boleh kosong pada sukses")
}

func TestUpdatePost_ForbiddenResponseBody(t *testing.T) {
	t.Parallel()

	svcMock := &mock.MockPostService{
		UpdateFn: func(_ context.Context, _ int64, _ domain.UpdatePostRequest, _ int64, _ string) (*domain.PostResponse, error) {
			return nil, domain.ErrForbidden
		},
	}

	router := newTestRouter(routerConfig{postSvc: svcMock})
	req := newJSONRequest(
		http.MethodPatch,
		"/api/v1/posts/7",
		domain.UpdatePostRequest{Title: strPtr("Coba Update")},
		withAuth(nonOwnerUserID, "user"),
	)

	w := executeRequest(router, req)

	require.Equal(t, http.StatusForbidden, w.Code)

	apiErr, err := decodeAPIError(w)
	require.NoError(t, err)

	assert.Equal(t, http.StatusForbidden, apiErr.Code)
	assert.NotEmpty(t, apiErr.Message, "pesan error tidak boleh kosong")
}
