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
		return err
	}

	log.Printf("Processing grade task: %s with payload: %+v", t.Type(), p)

	err := InsertGradeTaskFunc(db, p)
	if err != nil {
		log.Printf("Failed to insert grade task for %v: %v", p, err)
		return err
	}

	return nil
}
