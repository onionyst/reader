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
	"github.com/sirupsen/logrus"

	"reader/internal/app/reader/app"
	"reader/internal/app/reader/feeds"
	"reader/internal/app/reader/models"
	"reader/internal/app/reader/routes"
	"reader/internal/pkg/db"
	"reader/internal/pkg/logger"
	"reader/internal/pkg/middleware"
	pkgRoutes "reader/internal/pkg/routes"
	"reader/internal/pkg/utils"
)

const (
	httpAddr        = "0.0.0.0:3000"
	shutdownTimeout = 10 * time.Second
)

func main() {
	ctx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	utils.SetupTimeLocations()

	log := logger.New()
	log.Info("OnionReader")

	conns, err := db.Setup()
	if err != nil {
		log.Fatal(err)
	}
	defer conns.Close()

	if err := models.Register(conns.Main); err != nil {
		log.Fatal(err)
	}

	repo := models.NewRepo(conns.Main)

	_, stopFeeds := feeds.PollFeeds(ctx, log, repo)
	defer stopFeeds()

	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           setupRoutes(repo, log),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.WithField("addr", httpAddr).Info("http server starting")
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.WithError(err).Fatal("http server failed")
		}
	}()

	<-ctx.Done()
	log.Info("shutdown signal received")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.WithError(err).Warn("http server shutdown error")
	} else {
		log.Info("http server stopped")
	}
}

func setupRoutes(repo *models.Repo, log *logrus.Logger) *gin.Engine {
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.RequestLogger(log, "/healthz"))
	router.Use(middleware.Recovery(log))
	router.Use(middleware.ErrorResponder(log))
	router.GET("healthz", pkgRoutes.Healthz)

	routes.SetupRoutes(router, app.NewApp(repo))

	return router
}
