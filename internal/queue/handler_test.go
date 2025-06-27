package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"service_stats/internal/database"
	"service_stats/internal/model"
	"service_stats/internal/types"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
)

func TestHandleAddStadisticForStudent(t *testing.T) {
	// Setup mock
	mockCalled := false
	InsertGradeFunc = func(db *sql.DB, g model.Grade) error {
		mockCalled = true
		assert.Equal(t, "student1", g.StudentID)
		return nil
	}
	defer func() {
		// reset after test
		InsertGradeFunc = database.InsertGrade
	}()

	payload, _ := json.Marshal(model.Grade{StudentID: "student1", Grade: 90})

	task := asynq.NewTask(types.TaskAddStudentGrade, payload)

	err := HandleAddStadisticForStudent(context.Background(), task)
	assert.NoError(t, err)
	assert.True(t, mockCalled)
}

func TestHandleAddStadisticForStudent_BadJSON(t *testing.T) {
	task := asynq.NewTask(types.TaskAddStudentGrade, []byte("invalid json"))
	err := HandleAddStadisticForStudent(context.Background(), task)
	assert.Error(t, err)
}

func TestHandleAddStadisticForStudent_DBError(t *testing.T) {
	InsertGradeFunc = func(db *sql.DB, g model.Grade) error {
		return errors.New("db error")
	}
	defer func() {
		InsertGradeFunc = database.InsertGrade
	}()

	payload, _ := json.Marshal(model.Grade{StudentID: "student1", Grade: 90})
	task := asynq.NewTask(types.TaskAddStudentGrade, payload)

	err := HandleAddStadisticForStudent(context.Background(), task)
	assert.Error(t, err)
}

func TestHandleAddGradeTask(t *testing.T) {
	mockCalled := false
	InsertGradeTaskFunc = func(db *sql.DB, gt model.GradeTask) error {
		mockCalled = true
		assert.Equal(t, "task1", gt.TaskID)
		return nil
	}
	defer func() {
		InsertGradeTaskFunc = database.InsertGradeTask
	}()

	payload, _ := json.Marshal(model.GradeTask{TaskID: "task1", Grade: 85})

	task := asynq.NewTask(types.TaskAddStudentGradeTask, payload)

	err := HandleAddGradeTask(context.Background(), task)
	assert.NoError(t, err)
	assert.True(t, mockCalled)
}

func TestHandleAddGradeTask_BadJSON(t *testing.T) {
	task := asynq.NewTask(types.TaskAddStudentGradeTask, []byte("bad json"))
	err := HandleAddGradeTask(context.Background(), task)
	assert.Error(t, err)
}

func TestHandleAddGradeTask_DBError(t *testing.T) {
	InsertGradeTaskFunc = func(db *sql.DB, gt model.GradeTask) error {
		return errors.New("db error")
	}
	defer func() {
		InsertGradeTaskFunc = database.InsertGradeTask
	}()

	payload, _ := json.Marshal(model.GradeTask{TaskID: "task1", Grade: 85})
	task := asynq.NewTask(types.TaskAddStudentGradeTask, payload)

	err := HandleAddGradeTask(context.Background(), task)
	assert.Error(t, err)
}
