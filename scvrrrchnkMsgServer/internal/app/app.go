package app

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/config"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/db"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/handler"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/middleware"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/migrations"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/repository"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/service"
	"github.com/ValeriyOrlov/scvrrrchnkMsgServer/internal/ws"
	"github.com/gofiber/fiber/v2"
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

	hub := ws.NewHub()
	go hub.Run()

	chatRepo := repository.NewGormChatRepo(db)
	userRepo := repository.NewGormUserRepo(db)
	chatKeyRepo := repository.NewGormChatKeyRepo(db)
	userSevice := service.NewUserService(userRepo)
	chatService := service.NewChatService(chatRepo, userRepo, chatKeyRepo, db)
	chatHandler := handler.NewChatHandler(chatService, hub, userSevice, userRepo)
	messageRepo := repository.NewGormMessageRepo(db)
	messageService := service.NewMessageService(messageRepo, chatRepo, db, hub)
	messageHandler := handler.NewMessageHandler(messageService)
	userHandler := handler.NewUserHandler(userRepo)

	wsHandler := handler.NewWSHandler(hub, userRepo, chatService, messageService, cfg.JWTSecret)

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
		allowedOrigins = "http://localhost:5173"
	}

	fiberApp.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", allowedOrigins)
		c.Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Set("Access-Control-Allow-Methods", "GET, POST, HEAD, PUT, DELETE, PATCH")
		if c.Method() == "OPTIONS" {
			return c.SendStatus(204)
		}
		return c.Next()
	})
	api := fiberApp.Group("/api", middleware.AuthRequired(cfg.JWTSecret), middleware.EnsureUser(userRepo))
	api.Post("/chats", chatHandler.CreateChat)
	api.Get("/chats", chatHandler.GetUserChats)
	api.Get("/chats/:id", chatHandler.GetChatByID)
	api.Post("/chats/:id/messages", messageHandler.SendMessage)
	api.Get("/chats/:id/messages", messageHandler.GetMessages)
	api.Post("/chats/:id/members", chatHandler.AddMembers)
	api.Delete("/chats/:id/members", chatHandler.LeaveChat)
	api.Get("/users/search", chatHandler.SearchUsers)
	api.Patch("/chats/:chatId/messages/:messageId", messageHandler.UpdateMessage)
	api.Delete("/chats/:chatId/messages/:messageId", messageHandler.DeleteMessage)
	api.Get("/online", chatHandler.GetOnlineUsers)
	api.Get("/messages/:id", messageHandler.GetMessageByID)
	api.Put("/chats/:id/read", chatHandler.MarkAsRead)
	api.Put("/users/me/key", userHandler.UpdatePublicKey)
	api.Get("/chats/:id/keys", chatHandler.GetChatKeys)
	api.Post("/chats/:id/room-key", chatHandler.SaveRoomKey)
	api.Get("/chats/:id/room-key", chatHandler.GetRoomKey)
	api.Get("/users/:id/key", userHandler.GetPublicKey)
	api.Put("/users/me/backup", userHandler.SaveBackup)
	api.Get("/users/me/backup", userHandler.GetBackup)
	fiberApp.Get("/ws", wsHandler.HandleWebSocket())

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
