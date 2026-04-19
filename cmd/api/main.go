package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/dimasim/integrative-prog/internal/config"
	"github.com/dimasim/integrative-prog/internal/database"
	"github.com/dimasim/integrative-prog/internal/handler"
	"github.com/dimasim/integrative-prog/internal/middleware"
	"github.com/dimasim/integrative-prog/internal/repository"
	"github.com/dimasim/integrative-prog/internal/service"
)

func main() {
	// ── 1. Load Config ─────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// ── 2. Setup Logger ────────────────────────────────────────────────────
	setupLogger(cfg.LogLevel, cfg.App.Env)

	// ── 3. Connect Database ────────────────────────────────────────────────
	log.Info().Str("host", cfg.Database.Host).Msg("connecting to PostgreSQL...")
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()
	log.Info().Msg("database connected successfully")

	// ── 4. Wire Dependencies (manual DI: Repository → Service → Handler) ───
	userRepo := repository.NewUserRepository(db)
	userSvc  := service.NewUserService(userRepo, cfg.JWT)
	userHdlr := handler.NewUserHandler(userSvc)

	// ── 5. Setup Gin Router ────────────────────────────────────────────────
	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New() // Pakai gin.New(), bukan gin.Default() — custom middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestLogger())

	// Tangani 404 & 405 dengan format JSON yang konsisten
	router.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{"code": 405, "message": "Method not allowed"})
	})

	// Health check — tidak perlu auth
	router.GET("/health", func(c *gin.Context) {
		if err := db.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "db": "unreachable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "db": "connected"})
	})

	// ── 6. Register Routes ─────────────────────────────────────────────────
	jwtMW    := middleware.JWTAuth(cfg.JWT.Secret)
	apiKeyMW := middleware.APIKeyAuth(cfg.APIKey)
	adminMW  := middleware.RequireRole("admin")

	// /api/v1 → JWT-protected public API
	v1 := router.Group("/api/v1")
	userHdlr.RegisterRoutes(v1, jwtMW, adminMW)

	// /api/internal → API Key-protected (untuk service-to-service)
	internalGroup := router.Group("/api/internal", apiKeyMW)
	internalGroup.GET("/users", userHdlr.GetAll)

	// ── 7. Graceful Shutdown ───────────────────────────────────────────────
	srv := &http.Server{
		Addr:         ":" + cfg.App.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Info().Str("port", cfg.App.Port).Str("env", cfg.App.Env).Msg("🚀 server starting")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("server failed to start")
		}
	}()

	// Blokir hingga ada sinyal OS (Ctrl+C / docker stop)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("shutting down gracefully (max 10s)...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("forced shutdown")
	}
	log.Info().Msg("server exited cleanly ✓")
}

func setupLogger(level, env string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	if env != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: "15:04:05"})
	}
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)
}
