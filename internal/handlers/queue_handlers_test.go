package handlers

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"service_stats/internal/model"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type MockEnqueuer struct {
	EnqueueFunc func(taskType string, payload interface{}) (time.Duration, error)
}

func (m *MockEnqueuer) Enqueue(taskType string, payload interface{}) (time.Duration, error) {
	return m.EnqueueFunc(taskType, payload)
}

func TestEnqueueAddStadisticForStudent_Success(t *testing.T) {
	// Mock enqueuer returns 5 seconds delay, no error
	mock := &MockEnqueuer{
		EnqueueFunc: func(taskType string, payload interface{}) (time.Duration, error) {
			return 5 * time.Second, nil
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Use a sample payload
	payload := model.Grade{
		StudentID: "12345",
		CourseID:  "67890",
		Grade:     95,
	}

	// Call the handler
	EnqueueAddStadisticForStudent(c, mock, payload)

	// Check status code
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 but got %d", w.Code)
	}

	// Check response body contains success message
	expectedSubstring := "{\"result\":\"Task task:add_student_grade queued sucessfully (Expected time to be processed: 0.08 minutes)\",\"status\":200}"
	if !strings.Contains(w.Body.String(), expectedSubstring) {
		t.Errorf("expected body to contain %q, got %q", expectedSubstring, w.Body.String())
	}
}

func TestEnqueueAddStadisticForStudent_Error(t *testing.T) {
	// Mock enqueuer returns error
	mock := &MockEnqueuer{
		EnqueueFunc: func(taskType string, payload interface{}) (time.Duration, error) {
			return 0, errors.New("enqueue failed")
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	payload := model.Grade{
		StudentID: "12345",
		CourseID:  "67890",
		Grade:     95,
	}

	EnqueueAddStadisticForStudent(c, mock, payload)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 but got %d", w.Code)
	}

	if !strings.Contains(w.Body.String(), "Failed to enqueue task") {
		t.Errorf("expected body to contain failure message, got %q", w.Body.String())
	}
}
