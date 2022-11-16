package main

import (
	"os"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"

	"reader/internal/app/reader/db"
	"reader/internal/app/reader/feeds"
	"reader/internal/app/reader/routes"
	"reader/internal/pkg/utils"
)

const (
	serviceTimeout = 15 // seconds
)

// SetupRouter builds the router
func SetupRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.LoggerWithWriter(gin.DefaultWriter, "/ping"), gin.Recovery())
	routes.SetupRoutes(router)
	return router
}

func init() {
	utils.ResetRandom()
	utils.SetupTimeLocations()

	// logrus
	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)

	// services
	var services []string
	services = append(services, db.ServiceString())
	utils.Wait(services, serviceTimeout)
}

func main() {
	log.Info("OnionReader")

	pg := db.SetupDatabase()
	defer db.CloseDatabase(pg)

	feeds.LoadFeeds()

	router := SetupRouter()
	router.Run(":3000")
}
