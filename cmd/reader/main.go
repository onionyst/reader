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

	"reader/internal/app/reader/db"
	"reader/internal/app/reader/feeds"
	"reader/internal/app/reader/routes"
	"reader/internal/pkg/logging"
	"reader/internal/pkg/utils"
)

const (
	httpAddr        = "0.0.0.0:3000"
	serviceTimeout  = 15 // seconds
	shutdownTimeout = 10 * time.Second
)

func main() {
	ctx, stopSig := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSig()

	if err := utils.SetupTimeLocations(); err != nil {
		panic(err)
	}

	log := logging.Setup()
	log.Info("OnionReader")

	services := []string{db.ServiceString()}
	utils.Wait(services, serviceTimeout)

	pg := db.SetupDatabase()
	defer db.CloseDatabase(pg)

	_, stopFeeds := feeds.PollFeeds(ctx, log)
	defer stopFeeds()

	router := setupRouter(log)

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

func setupRouter(log *logrus.Logger) *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/healthz"))
	router.Use(logging.LogError(log))
	router.Use(gin.Recovery())

	routes.SetupRoutes(router)
	return router
}
