package handlers

import (
	"fmt"
	"net/http"
	"service_stats/internal/model"
	"service_stats/internal/queue"
	"service_stats/internal/types"

	"github.com/gin-gonic/gin"
)

func EnqueueAddStadisticForStudent(c *gin.Context, enqueuer *queue.Enqueuer, payload model.Grade) {
	taskType := types.TaskAddStudentGrade

	expected_delay_in_sec, err := enqueuer.Enqueue(taskType, payload)

	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"result": "Failed to enqueue task", "status": http.StatusBadRequest})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": fmt.Sprintf("Task %s queued sucessfully (Expected time to be processed: %.2f minutes)", taskType, expected_delay_in_sec.Minutes()), "status": http.StatusOK})
}
