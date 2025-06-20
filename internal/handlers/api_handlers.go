package handlers

import (
	"log"
	"net/http"
	"service_stats/internal/database"
	"service_stats/internal/model"

	"github.com/gin-gonic/gin"
)

func APIHandlerInsertGrade(c *gin.Context) {
	var grade model.Grade

	// Bind the JSON request to the Grade struct
	if err := c.ShouldBindJSON(&grade); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Invalid input", "status": http.StatusBadRequest})
		return
	}

	// Insert the grade into the database
	err := database.InsertGrade(grade)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"result": "Failed to insert grade", "status": http.StatusInternalServerError})
		return
	}

	log.Printf("Grade inserted successfully: %+v", grade)

	c.JSON(http.StatusOK, gin.H{"result": "Grade inserted successfully", "status": http.StatusOK})
}

func APIHandlerGetStatsForStudent(c *gin.Context) {
	studentID := c.Param("student_id")
	courseID := c.Param("course_id")

	if studentID == "" || courseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Missing student_id query parameter", "status": http.StatusBadRequest})
		return
	}

	avgGrade, err := database.GetAvgGradeForStudent(studentID, courseID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"result": "Failed to get average grade", "status": http.StatusInternalServerError})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": gin.H{
			"average_grade": avgGrade,
			"tbd":           0.0,
		},
		"course_id": courseID,
	})
}
