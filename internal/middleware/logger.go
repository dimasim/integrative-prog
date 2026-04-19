package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// RequestLogger adalah middleware zerolog untuk structured logging setiap request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		event := log.Info()
		if statusCode >= 500 {
			event = log.Error()
		} else if statusCode >= 400 {
			event = log.Warn()
		}

		event.
			Str("method", c.Request.Method).
			Str("path", path).
			Str("query", query).
			Int("status", statusCode).
			Dur("latency", duration).
			Str("ip", c.ClientIP()).
			Str("user_agent", c.Request.UserAgent()).
			Msg("request completed")
	}
}

// Recovery adalah middleware untuk menangkap panic dan mengembalikan 500.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		log.Error().
			Interface("panic", recovered).
			Str("path", c.Request.URL.Path).
			Msg("panic recovered")

		c.AbortWithStatusJSON(500, gin.H{
			"code":    500,
			"message": "Internal server error",
		})
	})
}
