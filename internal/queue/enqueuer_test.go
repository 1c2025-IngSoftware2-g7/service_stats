package queue

import (
	"errors"
	"testing"
	"time"

	"github.com/hibiken/asynq"
)

type MockAsynqClient struct {
	EnqueueFunc func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error)
}

func (m *MockAsynqClient) Enqueue(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
	if m.EnqueueFunc != nil {
		return m.EnqueueFunc(task, opts...)
	}
	return nil, errors.New("EnqueueFunc not implemented")
}

func TestEnqueue_Success(t *testing.T) {
	mockClient := &MockAsynqClient{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			// simulate successful enqueue
			return &asynq.TaskInfo{ID: "123"}, nil
		},
	}

	enqueuer := &Enqueuer{Client: mockClient}
	payload := struct {
		Name string
	}{Name: "test"}

	delay, err := enqueuer.Enqueue("test_task", payload)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if delay < 30*time.Second || delay > 180*time.Second {
		t.Errorf("expected delay between 30s and 180s, got %v", delay)
	}
}

func TestEnqueue_Failure(t *testing.T) {
	mockClient := &MockAsynqClient{
		EnqueueFunc: func(task *asynq.Task, opts ...asynq.Option) (*asynq.TaskInfo, error) {
			// simulate failure
			return nil, errors.New("enqueue failed")
		},
	}

	enqueuer := &Enqueuer{Client: mockClient}
	payload := struct {
		Name string
	}{Name: "test"}

	_, err := enqueuer.Enqueue("test_task", payload)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
