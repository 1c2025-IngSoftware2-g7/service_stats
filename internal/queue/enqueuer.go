package queue

import (
	"encoding/json"
	"log"
	"math/rand"
	"time"

	"github.com/hibiken/asynq"
)

/*type Enqueuer struct {
	Client *asynq.Client
}

func NewEnqueuer(redisAddr string) *Enqueuer {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &Enqueuer{Client: client}
}

func (e *Enqueuer) Enqueue(taskType string, payload interface{}) (time.Duration, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	task := asynq.NewTask(taskType, data)

	// lets set a random delay for the task to be processed
	min_seconds := 30
	max_seconds := 180 // 3 minutes

	// Randomly select a delay between 1 and 3 minutes
	delay := time.Duration(rand.Intn(max_seconds-min_seconds+1)+min_seconds) * time.Second

	_, err = e.Client.Enqueue(task, asynq.ProcessIn(delay))
	if err != nil {
		log.Printf("Failed to enqueue task: %v", err)
		return 0, err
	}

	log.Printf("Task enqueued: %s", taskType)

	// Lets convert the delay from seconds to minutes for the response
	return delay, nil
}
*/

type AsynqClient interface {
	Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

type Enqueuer struct {
	Client AsynqClient
}

func NewEnqueuer(redisAddr string) *Enqueuer {
	client := asynq.NewClient(asynq.RedisClientOpt{Addr: redisAddr})
	return &Enqueuer{Client: client}
}

func (e *Enqueuer) Enqueue(taskType string, payload interface{}) (time.Duration, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	task := asynq.NewTask(taskType, data)

	minSeconds := 30
	maxSeconds := 180
	delay := time.Duration(rand.Intn(maxSeconds-minSeconds+1)+minSeconds) * time.Second

	_, err = e.Client.Enqueue(task, asynq.ProcessIn(delay))
	if err != nil {
		log.Printf("Failed to enqueue task: %v", err)
		return 0, err
	}

	log.Printf("Task enqueued: %s", taskType)
	return delay, nil
}
