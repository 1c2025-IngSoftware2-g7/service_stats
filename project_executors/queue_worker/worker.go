package main

import (
	"fmt"
	"log"
	"os"

	"service_stats/internal/database"
	"service_stats/internal/queue"

	"github.com/hibiken/asynq"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables from .env file
	// err_env := godotenv.Load()
	// if err_env != nil {
	// 	log.Fatal("[Worker queue] Error loading .env file: ", err_env)
	// }

	database_url := os.Getenv("SERVICE_STATS_POSTGRES_URL")

	if database_url == "" {
		log.Fatal("[Stats Service] SERVICE_STATS_POSTGRES_URL environment variable is not set")
	}

	// Initialize the database connection with internal/database/db.go

	log.Printf("[Worker queue] Initializing database connection to [%s]", database_url)
	err := database.InitDB(database_url)

	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	ip_server := fmt.Sprintf("%s:%s", os.Getenv("ASYNC_QUEUE_HOST"), os.Getenv("ASYNC_QUEUE_PORT"))

	log.Printf("[Worker queue] Starting worker on {%s}", ip_server)
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: ip_server},
		asynq.Config{Concurrency: 10},
	)

	mux := queue.NewMux()

	if err := srv.Run(mux); err != nil {
		log.Fatalf("[Worker queue] Could not run worker: %v", err)
	}
}
