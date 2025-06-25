package model

import "time"

type GradeTask struct {
    StudentID string  `json:"student_id" binding:"required"`
    CourseID  string  `json:"course_id" binding:"required"`
    TaskID    string  `json:"task_id" binding:"required"`
    Grade     float64 `json:"grade" binding:"required"`
    OnTime    bool    `json:"on_time" binding:"required"`
    CreatedAt time.Time `json:"created_at"`
}