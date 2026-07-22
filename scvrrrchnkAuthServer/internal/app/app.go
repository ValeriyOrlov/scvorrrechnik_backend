package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/config"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/db"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/handler"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/migrations"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/repository"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	recoverware "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type App struct {
	fiberApp *fiber.App
	cfg      *config.Config
	db       *gorm.DB
	logger   *logrus.Logger
}

type logrusWriter struct {
	logger *logrus.Logger
}

func (w *logrusWriter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	w.logger.Info(message)
	return len(p), nil
}

func NewApp(cfg *config.Config) (*App, error) {
	appLogger := logrus.New()
	appLogger.SetFormatter(&logrus.JSONFormatter{})
	appLogger.SetLevel(logrus.InfoLevel)
	db, err := db.NewPostgresDB(cfg.DatabaseDSN, appLogger)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to database: %w", err)
	}

	if err := migrations.RunMigrations(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	userRepo := repository.NewGormUserRepo(db)
	tokenRepo := repository.NewGormTokenRepo(db)
	authService := service.NewAuthService(
		userRepo,
		tokenRepo,
		cfg.JWTSecret,
		cfg.AccessTokenTTL,
		cfg.RefreshTokenTTL,
	)
	authHandler := handler.NewAuthHandler(authService)

	fiberApp := fiber.New()
	fiberApp.Use(recoverware.New(recoverware.Config{
		EnableStackTrace: true,
	}))

	fiberApp.Use(logger.New(logger.Config{
		Output: &logrusWriter{logger: appLogger},
	}))
	log.Printf("ALLOW_ORIGINS is set to: %s", os.Getenv("ALLOW_ORIGINS"))

	allowedOrigins := os.Getenv("ALLOW_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "http://localhost:5173" // fallback для локальной разработки
	}

	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
		AllowMethods: "GET, POST, HEAD, PUT, DELETE, PATCH",
	}))
	fiberApp.Post("/register", authHandler.Register)
	fiberApp.Post("/login", authHandler.Login)
	fiberApp.Post("/refresh", authHandler.Refresh)
	fiberApp.Post("/logout", authHandler.Logout)

	fiberApp.Get("/me", handler.AuthRequired(cfg.JWTSecret), func(c *fiber.Ctx) error {
		userID := c.Locals("user_id")
		return c.JSON(fiber.Map{"user_id": userID})
	})

	return &App{
		fiberApp: fiberApp,
		cfg:      cfg,
		logger:   appLogger,
		db:       db,
	}, nil
}

func (a *App) Shutdown(ctx context.Context) error {
	if sqlDB, err := a.db.DB(); err == nil {
		if err := sqlDB.Close(); err != nil {
			a.logger.WithError(err).Error("db close error")
		}
	}
	return a.fiberApp.Shutdown()
}

func (a *App) Run() error {
	a.logger.Infof("Starting server on port %s", a.cfg.Port)
	return a.fiberApp.Listen(a.cfg.Port)
}
