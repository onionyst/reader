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
	"gorm.io/gorm"

	"reader/internal/app/reader/feeds"
	readerModels "reader/internal/app/reader/models"
	readerRoutes "reader/internal/app/reader/routes"
	"reader/internal/pkg/db"
	"reader/internal/pkg/logging"
	"reader/internal/pkg/utils"
)

const (
	httpAddr        = "0.0.0.0:3000"
	shutdownTimeout = 10 * time.Second
)

func main() {
	ctx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	if err := utils.SetupTimeLocations(); err != nil {
		panic(err)
	}

	// setup logger
	log := logging.Setup()
	log.Info("OnionReader")

	// setup database
	conns, err := db.Setup()
	if err != nil {
		log.Fatal(err)
	}
	defer conns.Close()

	if err := setupModels(conns.Main); err != nil {
		log.Fatal(err)
	}

	// emit workers
	_, stopFeeds := feeds.PollFeeds(ctx, log)
	defer stopFeeds()

	// setup routes
	router := setupRoutes(log)

	// launch server
	srv := &http.Server{
		Addr:              httpAddr,
		Handler:           router,
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

func setupModels(db *gorm.DB) error {
	if err := readerModels.Register(db); err != nil {
		return err
	}
	return nil
}

func setupRoutes(log *logrus.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/healthz"))
	router.Use(logging.LogError(log))
	router.Use(gin.Recovery())

	readerRoutes.SetupRoutes(router)
	return router
}
