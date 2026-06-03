package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/middleware"
	"github.com/dimasim/integrative-prog/internal/service"
)

// ── Handler structs ───────────────────────────────────────────────────────────

type PostHandler struct {
	postSvc    service.PostService
	commentSvc service.CommentService
}

func NewPostHandler(postSvc service.PostService, commentSvc service.CommentService) *PostHandler {
	return &PostHandler{
		postSvc:    postSvc,
		commentSvc: commentSvc,
	}
}

func (h *PostHandler) RegisterRoutes(rg *gin.RouterGroup, jwtAuth gin.HandlerFunc) {
	posts := rg.Group("/posts")
	{
		posts.GET("", h.ListPosts)
		posts.GET("/:id", h.GetPost)
		posts.GET("/:id/comments", h.ListComments)

		auth := posts.Group("", jwtAuth)
		{
			auth.POST("", h.CreatePost)
			auth.PATCH("/:id", h.UpdatePost)
			auth.DELETE("/:id", h.DeletePost)
			auth.POST("/:id/comments", h.CreateComment)
			auth.DELETE("/:id/comments/:comment_id", h.DeleteComment)
		}
	}
}

// ── Post handlers ─────────────────────────────────────────────────────────────

func (h *PostHandler) CreatePost(c *gin.Context) {
	var req domain.CreatePostRequest
	if !bindAndValidate(c, &req) {
		return
	}

	callerID := c.GetInt64(string(middleware.ContextKeyUserID))

	resp, err := h.postSvc.Create(c.Request.Context(), req, callerID)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{
		Code:    http.StatusCreated,
		Message: "post created",
		Data:    resp,
	})
}

func (h *PostHandler) GetPost(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}

	resp, svcErr := h.postSvc.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    resp,
	})
}

func (h *PostHandler) ListPosts(c *gin.Context) {
	var req domain.PostListRequest
	_ = c.ShouldBindQuery(&req)
	if req.Limit <= 0 {
		req.Limit = 10
	}

	resp, err := h.postSvc.List(c.Request.Context(), req.Limit, req.Offset)
	if err != nil {
		handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    resp,
	})
}

func (h *PostHandler) UpdatePost(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}

	var req domain.UpdatePostRequest
	if !bindAndValidate(c, &req) {
		return
	}

	callerID := c.GetInt64(string(middleware.ContextKeyUserID))
	callerRole := c.GetString(string(middleware.ContextKeyRole))

	resp, svcErr := h.postSvc.Update(c.Request.Context(), id, req, callerID, callerRole)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "post updated",
		Data:    resp,
	})
}

func (h *PostHandler) DeletePost(c *gin.Context) {
	id, err := parseID(c, "id")
	if err != nil {
		return
	}

	callerID := c.GetInt64(string(middleware.ContextKeyUserID))
	callerRole := c.GetString(string(middleware.ContextKeyRole))

	if svcErr := h.postSvc.Delete(c.Request.Context(), id, callerID, callerRole); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "post deleted",
	})
}

// ── Comment handlers ──────────────────────────────────────────────────────────

func (h *PostHandler) CreateComment(c *gin.Context) {
	postID, err := parseID(c, "id")
	if err != nil {
		return
	}

	var req domain.CreateCommentRequest
	if !bindAndValidate(c, &req) {
		return
	}

	callerID := c.GetInt64(string(middleware.ContextKeyUserID))

	resp, svcErr := h.commentSvc.Create(c.Request.Context(), postID, req, callerID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusCreated, domain.APIResponse{
		Code:    http.StatusCreated,
		Message: "comment created",
		Data:    resp,
	})
}

func (h *PostHandler) ListComments(c *gin.Context) {
	postID, err := parseID(c, "id")
	if err != nil {
		return
	}

	resp, svcErr := h.commentSvc.ListByPost(c.Request.Context(), postID)
	if svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "ok",
		Data:    resp,
	})
}

func (h *PostHandler) DeleteComment(c *gin.Context) {
	_, err := parseID(c, "id")
	if err != nil {
		return
	}

	commentID, err := parseID(c, "comment_id")
	if err != nil {
		return
	}

	callerID := c.GetInt64(string(middleware.ContextKeyUserID))
	callerRole := c.GetString(string(middleware.ContextKeyRole))

	if svcErr := h.commentSvc.Delete(c.Request.Context(), commentID, callerID, callerRole); svcErr != nil {
		handleServiceError(c, svcErr)
		return
	}

	c.JSON(http.StatusOK, domain.APIResponse{
		Code:    http.StatusOK,
		Message: "comment deleted",
	})
}