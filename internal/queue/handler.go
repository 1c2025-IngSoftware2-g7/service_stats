package queue

import (
	"context"
	"encoding/json"
	"log"

	"service_stats/internal/database"
	"service_stats/internal/model"
	"service_stats/internal/types"

	// Add this line to import the internal package
	"github.com/hibiken/asynq"
)

func NewMux() *asynq.ServeMux {
    mux := asynq.NewServeMux()
    mux.HandleFunc(types.TaskAddStudentGrade, HandleAddStadisticForStudent)
    mux.HandleFunc(types.TaskAddStudentGradeTask, HandleAddGradeTask)
    return mux
}

func HandleAddStadisticForStudent(ctx context.Context, t *asynq.Task) error {
	var p model.Grade
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		return err
	}

	log.Printf("Processing task: %s with payload: %+v", t.Type(), p)

	err := database.InsertGrade(p)

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

    err := database.InsertGradeTask(p)
    if err != nil {
        log.Printf("Failed to insert grade task for %v: %v", p, err)
        return err
    }

    return nil
}