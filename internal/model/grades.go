package model

type Grade struct {
	StudentID string  `json:"student_id" binding:"required"`
	CourseID  string  `json:"course_id" binding:"required"`
	Grade     float64 `json:"grade" binding:"required"`
	OnTime    bool    `json:"on_time" binding:"required"`
}
