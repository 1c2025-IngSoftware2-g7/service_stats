package handlers

import (
	"log"
	"net/http"
	"service_stats/internal/database"
	"service_stats/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func APIHandlerInsertGrade( /*c *gin.Context*/ StudentGrade model.Grade) {

	err := database.InsertGrade(StudentGrade)

	if err != nil {
		// Handle the error, e.g., log it and return an error response
		log.Printf("Error inserting grade with data %v: %v", StudentGrade, err)
		return
	}

	/*
		Lo dejo como schema para el futuro
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

		c.JSON(http.StatusOK, gin.H{"result": "Grade inserted successfully", "status": http.StatusOK})*/
}

func APIHandlerGetStatsForStudent(c *gin.Context) {
	studentID := c.Param("student_id")
	courseID := c.Param("course_id")

	if studentID == "" || courseID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Missing student_id query parameter", "status": http.StatusBadRequest})
		return
	}

	// Lets check if studenID and courseID are a valid UUID
	if _, err := uuid.Parse(studentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Invalid student_id format (not an UUID value)", "status": http.StatusBadRequest})
	}

	if _, err := uuid.Parse(courseID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Invalid course_id format (not an UUID value)", "status": http.StatusBadRequest})
	}

	avgGrade, err, code := database.GetAvgGradeForStudent(studentID, courseID)
	if err != nil {
		c.JSON(code, gin.H{"result": "Failed to get average grade", "status": http.StatusInternalServerError})
		return
	}

	if code == http.StatusNotFound {
		c.JSON(code, gin.H{"result": "No grades found for the student in the course", "status": http.StatusNotFound})
		return
	}

	c.JSON(code, gin.H{
		"result": gin.H{
			"average_grade": avgGrade,
			"tbd":           0.0,
		},
		"course_id": courseID,
	})
}
