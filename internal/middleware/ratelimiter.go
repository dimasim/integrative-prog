package middleware

import (
	"net/http"
	"sync"
	// Hapus import "time" karena tidak digunakan

	"github.com/dimasim/integrative-prog/internal/domain" // Sesuaikan nama modul
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.RWMutex
	r   rate.Limit
	b   int
}

func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.RWMutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]
	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

func RateLimit() gin.HandlerFunc {
	// 5 request per detik, maksimal burst 10
	limiter := NewIPRateLimiter(5, 10)

	return func(c *gin.Context) {
		ip := c.ClientIP()
		limiterForIP := limiter.GetLimiter(ip)

		if !limiterForIP.Allow() {
			// PERBAIKAN: Ubah Code menjadi integer (429) menggunakan konstanta bawaan Go
			errResp := domain.APIError{
				Code:    http.StatusTooManyRequests, 
				Message: "Terlalu banyak request, silakan coba lagi nanti.",
			}
			c.AbortWithStatusJSON(http.StatusTooManyRequests, domain.APIResponse{
				Code:    http.StatusTooManyRequests,
				Message: "Too Many Requests",
				Data:    errResp,
			})
			return
		}
		c.Next()
	}
}