package model

import "time"

type Grade struct {
	StudentID string    `json:"student_id" binding:"required"`
	CourseID  string    `json:"course_id" binding:"required"`
	Grade     float64   `json:"grade" binding:"required"`
	OnTime    bool      `json:"on_time"`
	CreatedAt time.Time `json:"created_at"`
}

func NewGrade(studentID, courseID string, grade float64, onTime bool) Grade {
	return Grade{
		StudentID: studentID,
		CourseID:  courseID,
		Grade:     grade,
		OnTime:    onTime,
		CreatedAt: time.Now(),
	}
}
