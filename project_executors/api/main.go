package main

import (
	"log"
	"net/http"
	"service_stats/internal/monitoring"

	"github.com/gin-gonic/gin"
	"github.com/newrelic/go-agent/v3/newrelic"
	//"github.com/joho/godotenv"
)

func setupRouter(nrApp monitoring.Application) *gin.Engine {
	router := gin.Default()

	router.Use(func(c *gin.Context) {
		txn := nrApp.StartTransaction(c.FullPath())
		defer txn.End()

		c.Set("newrelic.Transaction", txn)
		c.Next()
	})

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	return router
}
func main() {
	newRelicApp, err := newrelic.NewApplication(
		newrelic.ConfigAppName("MyApp"),
		newrelic.ConfigLicense("your_license_key_here"),
	)
	if err != nil {
		log.Fatal("New Relic initialization error:", err)
	}

	appWrapper := &monitoring.NewRelicApp{
		App: newRelicApp,
	}

	router := setupRouter(appWrapper)

	log.Println("Starting server on :8080")
	router.Run(":8080")
}
