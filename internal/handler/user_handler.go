package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/service"
)

var validate = validator.New()

type UserHandler struct {
	svc service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// RegisterRoutes mendaftarkan semua route user ke router group.
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup, authMW, adminMW gin.HandlerFunc) {
	// Public — tidak butuh auth
	rg.POST("/auth/register", h.Register)
	rg.POST("/auth/login", h.Login)

	// Protected — butuh JWT
	users := rg.Group("/users", authMW)
	{
		users.GET("", adminMW, h.GetAll)      // hanya admin
		users.GET("/:id", h.GetByID)          // semua authenticated user
		users.PATCH("/:id", adminMW, h.Update) // hanya admin
		users.DELETE("/:id", adminMW, h.Delete) // hanya admin
	}
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// Register godoc
// @Summary Register user baru
// @Tags auth
// @Accept json
// @Produce json
// @Param body body domain.CreateUserRequest true "User payload"
// @Success 201 {object} domain.APIResponse
// @Failure 400 {object} domain.APIError
// @Failure 409 {object} domain.APIError
// @Router /auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req domain.CreateUserRequest
	if !bindAndValidate(c, &req) {
		return
	}

	user, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{
		Code:    http.StatusCreated,
		Message: "User registered successfully",
		Data:    user,
	})
}

// Login godoc
// @Summary Login dan dapatkan JWT token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body domain.LoginRequest true "Login credentials"
// @Success 200 {object} domain.APIResponse
// @Failure 400 {object} domain.APIError
// @Failure 401 {object} domain.APIError
// @Router /auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req domain.LoginRequest
	if !bindAndValidate(c, &req) {
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "Login successful",
		Data:    resp,
	})
}

func (h *UserHandler) GetAll(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	users, err := h.svc.GetAll(c.Request.Context(), limit, offset)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "Users retrieved successfully",
		Data:    users,
	})
}

func (h *UserHandler) GetByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	user, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "User retrieved successfully",
		Data:    user,
	})
}

func (h *UserHandler) Update(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	var req domain.UpdateUserRequest
	if !bindAndValidate(c, &req) {
		return
	}

	user, err := h.svc.Update(c.Request.Context(), id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "User updated successfully",
		Data:    user,
	})
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		return
	}

	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "User deleted successfully",
	})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// bindAndValidate bind JSON body dan validasi menggunakan struct tags.
// Mengembalikan false jika ada error dan sudah menulis response.
func bindAndValidate(c *gin.Context, req interface{}) bool {
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body: " + err.Error(),
		})
		return false
	}
	if err := validate.Struct(req); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, domain.APIError{
				Code:    http.StatusBadRequest,
				Message: formatValidationErrors(ve),
			})
			return false
		}
	}
	return true
}

// handleServiceError — peta sentinel errors ke HTTP status codes.
// Tidak pernah bocorkan detail DB atau stack trace ke client.
func handleServiceError(c *gin.Context, err error) {
	log.Err(err).Str("path", c.Request.URL.Path).Msg("service error")

	switch {
	case errors.Is(err, domain.ErrNotFound):
		c.JSON(http.StatusNotFound, domain.APIError{Code: 404, Message: "Resource not found"})
	case errors.Is(err, domain.ErrConflict):
		c.JSON(http.StatusConflict, domain.APIError{Code: 409, Message: "Resource already exists"})
	case errors.Is(err, domain.ErrInvalidCreds):
		c.JSON(http.StatusUnauthorized, domain.APIError{Code: 401, Message: "Invalid credentials"})
	case errors.Is(err, domain.ErrForbidden):
		c.JSON(http.StatusForbidden, domain.APIError{Code: 403, Message: "Access denied"})
	default:
		c.JSON(http.StatusInternalServerError, domain.APIError{Code: 500, Message: "Internal server error"})
	}
}

func parseID(c *gin.Context) (int64, error) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, domain.APIError{
			Code:    http.StatusBadRequest,
			Message: "Invalid ID parameter",
		})
		return 0, err
	}
	return id, nil
}

func formatValidationErrors(ve validator.ValidationErrors) string {
	msg := "Validation failed:"
	for _, fe := range ve {
		msg += " [" + fe.Field() + ": " + fe.Tag() + "]"
	}
	return msg
}
