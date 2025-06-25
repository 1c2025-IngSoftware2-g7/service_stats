package handlers

import (
	"encoding/hex"
	"log"
	"net/http"
	"service_stats/internal/database"
	"service_stats/internal/model"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func isValidObjectID(id string) bool {
	if len(id) != 24 {
		return false
	}

	_, err := hex.DecodeString(id)

	return err == nil
}

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
		return
	}

	if !isValidObjectID(courseID) {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Invalid course_id format (not a valid ObjectID)", "status": http.StatusBadRequest})
		return
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


// Estructuras para las requests
type TimeRangeRequest struct {
    StartDate  string `form:"start_date"`
    EndDate    string `form:"end_date"`
    GroupBy    string `form:"group_by"` // "day", "week", "month", "quarter", "year"
}

// Helper function para parsear fechas
func parseTimeRange(start, end string) (time.Time, time.Time, error) {
    layout := "2006-01-02" // Formato YYYY-MM-DD
    var startTime, endTime time.Time
    var err error

    if start != "" {
        startTime, err = time.Parse(layout, start)
        if err != nil {
            return time.Time{}, time.Time{}, err
        }
    } else {
        startTime = time.Time{} // Cero value para indicar "sin límite"
    }

    if end != "" {
        endTime, err = time.Parse(layout, end)
        if err != nil {
            return time.Time{}, time.Time{}, err
        }
        // Ajustamos para incluir todo el día final
        endTime = endTime.Add(24 * time.Hour - time.Nanosecond)
    } else {
        endTime = time.Now()
    }

    return startTime, endTime, nil
}

// Handler para promedio de estudiante
func APIHandlerGetStudentAverageOverTime(c *gin.Context) {
    studentID := c.Param("student_id")
    var req TimeRangeRequest
    
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    startTime, endTime, err := parseTimeRange(req.StartDate, req.EndDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
        return
    }

    averages, err := database.GetStudentAveragesOverTime(studentID, startTime, endTime, req.GroupBy)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "student_id": studentID,
        "averages":   averages,
        "time_range": gin.H{
            "start": startTime.Format(time.RFC3339),
            "end":   endTime.Format(time.RFC3339),
        },
        "group_by": req.GroupBy,
    })
}

// Handler para promedio de curso (similar al anterior pero para cursos)
func APIHandlerGetCourseAverageOverTime(c *gin.Context) {
    courseID := c.Param("course_id")
    var req TimeRangeRequest
    
    if err := c.ShouldBindQuery(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
        return
    }

    startTime, endTime, err := parseTimeRange(req.StartDate, req.EndDate)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format. Use YYYY-MM-DD"})
        return
    }

    averages, err := database.GetCourseAveragesOverTime(courseID, startTime, endTime, req.GroupBy)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    c.JSON(http.StatusOK, gin.H{
        "course_id": courseID,
        "averages":   averages,
        "time_range": gin.H{
            "start": startTime.Format(time.RFC3339),
            "end":   endTime.Format(time.RFC3339),
        },
        "group_by": req.GroupBy,
    })
}