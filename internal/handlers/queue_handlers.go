package handlers

import (
	"fmt"
	"net/http"
	"service_stats/internal/model"
	"service_stats/internal/types"
	"time"

	"github.com/gin-gonic/gin"
)

type Enqueuer interface {
	Enqueue(taskType string, payload interface{}) (time.Duration, error)
}

func EnqueueAddStadisticForStudent(c *gin.Context, enqueuer Enqueuer, payload model.Grade) {
	taskType := types.TaskAddStudentGrade

	expected_delay_in_sec, err := enqueuer.Enqueue(taskType, payload)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Failed to enqueue task", "status": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": fmt.Sprintf("Task %s queued sucessfully (Expected time to be processed: %.2f minutes)", taskType, expected_delay_in_sec.Minutes()), "status": http.StatusOK})
}

func EnqueueAddGradeTask(c *gin.Context, enqueuer Enqueuer, payload model.GradeTask) {
	taskType := types.TaskAddStudentGradeTask

	expected_delay_in_sec, err := enqueuer.Enqueue(taskType, payload)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Failed to enqueue task", "status": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"result": fmt.Sprintf("Task %s queued successfully (Expected time to be processed: %.2f minutes)",
			taskType, expected_delay_in_sec.Minutes()),
		"status": http.StatusOK,
	})
}
