package main

import (
	"log"
	"os"
	"service_stats/internal/database"
	"service_stats/internal/handlers"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {

	// Load environment variables from .env file
	err_env := godotenv.Load()

	if err_env != nil {
		log.Fatal("[Stats Service] Error loading .env file: ", err_env)
	}

	// Initialize New Relic
	newRelicApp, err_relic := newrelic.NewApplication(
		newrelic.ConfigAppName(os.Getenv("NEW_RELIC_APP_NAME")),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE_KEY")),
		newrelic.ConfigDistributedTracerEnabled(true),
		func(c *newrelic.Config) {
			c.Enabled = true
		},
	)
	if err_relic != nil {
		log.Fatal("[Stats Service] Error initializing New Relic: ", err_relic)
	}

	database_url := os.Getenv("SERVICE_STATS_POSTGRES_URL")

	if database_url == "" {
		log.Fatal("[Stats Service] SERVICE_STATS_POSTGRES_URL environment variable is not set")
	}

	// Initialize the database connection with internal/database/db.go
	err := database.InitDB(database_url)

	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	router := gin.Default()

	router.Use(func(c *gin.Context) {
		// Start a new New Relic transaction
		txn := newRelicApp.StartTransaction(c.FullPath())

		defer txn.End()

		// Set the transaction in the context
		c.Set("newrelic.Transaction", txn)

		// Continue with the next handler
		c.Next()
	})

	{
		routing := router.Group("/stats")

		routing.GET("/health", handlers.HealthCheckHandler)

		routing.POST("/student/grade", handlers.APIHandlerInsertGrade)
		routing.GET("/average/", handlers.APIHandlerGetAvgGradeForStudent)
	}

	// Lets log the server start
	logger := log.New(gin.DefaultWriter, "INFO: ", log.LstdFlags)

	logger.Println("Server started on port 8080")

	router.Run("0.0.0.0:8080")
}
