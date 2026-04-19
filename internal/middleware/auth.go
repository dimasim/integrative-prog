package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"

	"github.com/dimasim/integrative-prog/internal/domain"
	"github.com/dimasim/integrative-prog/internal/service"
)

const (
	ContextKeyUserID = "user_id"
	ContextKeyRole   = "role"
	ContextKeyEmail  = "email"
)

// JWTAuth adalah middleware untuk autentikasi JWT Bearer Token.
// Cara pakai: router.Use(middleware.JWTAuth(jwtSecret))
func JWTAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondUnauthorized(c, "Authorization header is required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			respondUnauthorized(c, "Authorization header format must be: Bearer <token>")
			return
		}

		tokenStr := parts[1]
		claims := &service.JWTClaims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			// Validasi signing method — cegah algorithm confusion attack
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(jwtSecret), nil
		})

		if err != nil || !token.Valid {
			log.Warn().Err(err).Str("ip", c.ClientIP()).Msg("invalid JWT token")
			respondUnauthorized(c, "Invalid or expired token")
			return
		}

		// Simpan claims ke context untuk diakses handler
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyRole, claims.Role)
		c.Next()
	}
}

// APIKeyAuth adalah middleware untuk autentikasi API Key (service-to-service).
// Key dibaca dari header X-API-Key.
func APIKeyAuth(apiKey string) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			respondUnauthorized(c, "X-API-Key header is required")
			return
		}
		// Perbandingan string yang aman (constant time tidak diperlukan untuk API Key sederhana,
		// tapi untuk produksi gunakan subtle.ConstantTimeCompare)
		if key != apiKey {
			log.Warn().Str("ip", c.ClientIP()).Msg("invalid API key attempt")
			respondUnauthorized(c, "Invalid API key")
			return
		}
		c.Next()
	}
}

// RequireRole memastikan user yang terautentikasi memiliki role tertentu.
// Wajib digunakan SETELAH JWTAuth.
func RequireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get(ContextKeyRole)
		if !exists {
			respondForbidden(c, "Access denied")
			return
		}
		for _, r := range roles {
			if r == userRole.(string) {
				c.Next()
				return
			}
		}
		log.Warn().
			Str("user_role", userRole.(string)).
			Strs("required_roles", roles).
			Msg("insufficient role")
		respondForbidden(c, "Insufficient permissions")
	}
}

// ─── Helper ───────────────────────────────────────────────────────────────────

func respondUnauthorized(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, domain.APIError{
		Code:    http.StatusUnauthorized,
		Message: msg,
	})
}

func respondForbidden(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusForbidden, domain.APIError{
		Code:    http.StatusForbidden,
		Message: msg,
	})
}
