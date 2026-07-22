package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/app"
	"github.com/ValeriyOrlov/scvrrrchnkAuthServer/internal/config"
	"github.com/sirupsen/logrus"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		logrus.WithError(err).Fatalf("cannot load config: %v", err)
	}
	application, err := app.NewApp(cfg)
	if err != nil {
		logrus.Fatal(err)
	}
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := application.Run(); err != nil {
			logrus.WithError(err).Info("server stopped")
		}
	}()

	sig := <-quit
	logrus.Infof("recevied signal: %v, shutting down gracefully", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx); err != nil {
		logrus.WithError(err).Fatal("shutdown failed")
	}
	logrus.Info("server exited cleanly")
}
