package queue

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"

	"service_stats/internal/database"
	"service_stats/internal/model"
	"service_stats/internal/types"

	// Add this line to import the internal package
	"github.com/hibiken/asynq"
)

var db *sql.DB

func NewMux(database_ref *sql.DB) *asynq.ServeMux {
	mux := asynq.NewServeMux()
	mux.HandleFunc(types.TaskAddStudentGrade, HandleAddStadisticForStudent)
	mux.HandleFunc(types.TaskAddStudentGradeTask, HandleAddGradeTask)
	db = database_ref
	return mux
}

var (
	InsertGradeFunc     = database.InsertGrade
	InsertGradeTaskFunc = database.InsertGradeTask
)

func HandleAddStadisticForStudent(ctx context.Context, t *asynq.Task) error {
	var p model.Grade
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	log.Printf("Processing task: %s with payload: %+v", t.Type(), p)

	err := InsertGradeFunc(db, p)
	if err != nil {
		log.Printf("Failed to insert grade for %v: %v", p, err)
		return err
	}

	return nil
}

func HandleAddGradeTask(ctx context.Context, t *asynq.Task) error {
	var p model.GradeTask
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		log.Printf("[ERROR] Failed to unmarshal task payload: %v", err)
		return err
	}

	log.Printf("Processing grade task: %s with payload: %+v", t.Type(), p)

	exists, err := database.CheckGradeTaskExists(p.StudentID, p.CourseID, p.TaskID)
	if err != nil {
		log.Printf("[ERROR] Checking grade task existence: %v", err)
		return err
	}

	if exists {
		err = database.UpdateGradeTask(p)
		if err != nil {
			log.Printf("[ERROR] Updating grade task: %v", err)
			return err
		}
		log.Printf("Grade task UPDATED - Student: %s, Course: %s, Task: %s, Grade: %.2f, OnTime: %t",
			p.StudentID, p.CourseID, p.TaskID, p.Grade, p.OnTime)
	} else {
		err = database.InsertGradeTask(p)
		if err != nil {
			log.Printf("[ERROR] Inserting grade task: %v", err)
			return err
		}
		log.Printf("Grade task INSERTED - Student: %s, Course: %s, Task: %s, Grade: %.2f, OnTime: %t",
			p.StudentID, p.CourseID, p.TaskID, p.Grade, p.OnTime)
	}

	return nil
}
