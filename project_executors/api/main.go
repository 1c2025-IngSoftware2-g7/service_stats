package main

import (
	"fmt"
	"log"
	"os"
	"service_stats/internal/database"
	"service_stats/internal/handlers"
	"service_stats/internal/model"
	"service_stats/internal/queue"

	"github.com/gin-gonic/gin"
	//"github.com/joho/godotenv"
	"github.com/newrelic/go-agent/v3/newrelic"
)

func main() {

	// Load environment variables from .env file
	// err_env := godotenv.Load()

	// if err_env != nil {
	// 	log.Fatal("[Stats Service] Error loading .env file: ", err_env)
	// }

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

	server_ip := fmt.Sprintf("%s:%s", os.Getenv("ASYNC_QUEUE_HOST"), os.Getenv("ASYNC_QUEUE_PORT"))
	enqueuer := queue.NewEnqueuer(server_ip)

	database_url := os.Getenv("SERVICE_STATS_POSTGRES_URL")

	if database_url == "" {
		log.Fatal("[Stats Service] SERVICE_STATS_POSTGRES_URL environment variable is not set")
	}

	// Initialize the database connection with internal/database/db.go

	log.Printf("[Main APP] Initializing database connection to [%s]", database_url)
	db_ref, err_creating := database.InitDB(database_url)

	if err_creating != nil {
		log.Fatalf("Failed to initialize database: %v", err_creating)
	}

	{
		routing := router.Group("/stats")

		routing.GET("/health", handlers.HealthCheckHandler)

		//routing.POST("/student/grade", handlers.APIHandlerInsertGrade)

		// For each POST, we will enqueue a task to process the student grade
		routing.POST("/student/grade", func(c *gin.Context) {
			var grade model.Grade
			if err := c.ShouldBindJSON(&grade); err != nil {
				c.JSON(400, gin.H{"error": "Invalid input"})
				return
			}

			// Enqueue the task to add student grade
			handlers.EnqueueAddStadisticForStudent(c, enqueuer, grade)
		})

		routing.GET("/student/:student_id/course/:course_id", func(c *gin.Context) {
			handlers.APIHandlerGetStatsForStudent(db_ref, c)
		})

		// Endpoints individuales
		routing.GET("/student/:student_id/average", func(c *gin.Context) {
			handlers.APIHandlerGetStudentAverageOverTime(db_ref, c)
		})
		routing.GET("/course/:course_id/average", func(c *gin.Context) {
			handlers.APIHandlerGetCourseAverageOverTime(db_ref, c)
		})

		routing.POST("/student/task/grade", func(c *gin.Context) {
			var gradeTask model.GradeTask
			if err := c.ShouldBindJSON(&gradeTask); err != nil {
				c.JSON(400, gin.H{"error": "Invalid input"})
				return
			}
			handlers.EnqueueAddGradeTask(c, enqueuer, gradeTask)

			//routing.GET("/student/:student_id/course/:course_id/task/:task_id", handlers.APIHandlerGetStatsForStudentTask)
		})

		routing.GET("/student/:student_id/course/:course_id/task/average", func(c *gin.Context) {
			handlers.APIHandlerGetStudentCourseTasksAverage(db_ref, c)
		})

		routing.GET("/course/:course_id/task/:task_id/averages", func(c *gin.Context) {
			handlers.APIHandlerGetTaskAverages(db_ref, c)
		})

		routing.GET("/course/:course_id/on_time_percentage", handlers.APIHandlerGetCourseOnTimePercentage)

		routing.GET("/course/:course_id/student/:student_id/on_time_percentage", handlers.APIHandlerGetStudentOnTimePercentage)
    })
	}

	// Lets log the server start
	logger := log.New(gin.DefaultWriter, "INFO: ", log.LstdFlags)

	logger.Println("Server started on port 8080")

	router.Run("0.0.0.0:8080")
}
