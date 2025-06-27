package handlers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthCheckHandler(t *testing.T) {
	w := httptest.NewRecorder()

	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest("GET", "/health", nil)

	HealthCheckHandler(c)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200 but got %d", w.Code)
	}

	// Assert the response body contains the expected JSON
	expectedBody := `{"status":"OK"}`
	if strings.TrimSpace(w.Body.String()) != expectedBody {
		t.Fatalf("expected body %s but got %s", expectedBody, w.Body.String())
	}
}
