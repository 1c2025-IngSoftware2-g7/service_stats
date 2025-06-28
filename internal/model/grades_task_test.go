package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGradeTask(t *testing.T) {
	gradeTask := NewGradeTask("student1", "course1", "task1", 90.0, true)

	assert.Equal(t, "student1", gradeTask.StudentID)
	assert.Equal(t, "course1", gradeTask.CourseID)
	assert.Equal(t, "task1", gradeTask.TaskID)
	assert.Equal(t, 90.0, gradeTask.Grade)
	assert.Equal(t, true, gradeTask.OnTime)
	assert.NotZero(t, gradeTask.CreatedAt) // Check that CreatedAt is set to a non-zero value
}
