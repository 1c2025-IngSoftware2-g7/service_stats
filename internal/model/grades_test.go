package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGrade(t *testing.T) {
	grade := NewGrade("student1", "course1", 85.0, true)

	assert.Equal(t, "student1", grade.StudentID)
	assert.Equal(t, "course1", grade.CourseID)
	assert.Equal(t, 85.0, grade.Grade)
	assert.Equal(t, true, grade.OnTime)
}
