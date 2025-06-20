package main

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	router := gin.Default()

	{
		routing := router.Group("/stats/")

		routing.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "OK"})
		})
	}

	// Lets start the server
	router.Run("0.0.0.0:8080")

	// Lets log the server start
	logger := log.New(gin.DefaultWriter, "INFO: ", log.LstdFlags)

	logger.Println("Server started on port 8080")

	// lets print
	print("Server started on port 8080\n")
}
