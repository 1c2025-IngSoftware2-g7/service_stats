package main

import (
	"net/http"
	"net/http/httptest"
	"service_stats/internal/monitoring"
	"sync"
	"testing"
)

type MockTransaction struct {
	mu    sync.Mutex
	ended bool
}

func (m *MockTransaction) End() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ended = true
}

func (m *MockTransaction) Ended() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ended
}

type MockApplication struct {
	mu                  sync.Mutex
	lastTransactionName string
	lastTransaction     *MockTransaction
}

func (m *MockApplication) StartTransaction(name string) monitoring.Transaction {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.lastTransactionName = name
	m.lastTransaction = &MockTransaction{}
	return m.lastTransaction
}

func (m *MockApplication) LastTransactionName() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastTransactionName
}

func (m *MockApplication) TransactionEnded() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastTransaction == nil {
		return false
	}
	return m.lastTransaction.ended
}

// --- Step 6: Test that uses the mock ---

func TestRouterWithMockNewRelic(t *testing.T) {
	mockRelic := &MockApplication{}

	router := setupRouter(mockRelic)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200 but got %d", w.Code)
	}

	if got := mockRelic.LastTransactionName(); got != "/health" {
		t.Errorf("Expected transaction name '/health' but got '%s'", got)
	}

	if !mockRelic.TransactionEnded() {
		t.Error("Expected transaction to be ended but it was not")
	}
}
