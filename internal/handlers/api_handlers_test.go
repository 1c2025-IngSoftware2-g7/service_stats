package handlers

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"service_stats/internal/database"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIsValidObjectID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"507f1f77bcf86cd799439011", true},
		{"507f1f77bcf86cd79943901", true},
		{"507f1f77bcf86cd7994390111", true},
		{"invalid_id", false},
	}

	for _, test := range tests {
		result := isValidObjectID(test.id)
		if result != test.expected {
			t.Errorf("isValidObjectID(%s) = %v; expected %v", test.id, result, test.expected)
		}
	}
}

func TestIsValidObjectID_MultipleTests(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Length 1 valid letter",
			input:    "a",
			expected: true,
		},
		{
			name:     "Length 1 valid number",
			input:    "5",
			expected: true,
		},
		{
			name:     "Length 1 invalid char",
			input:    "-",
			expected: false,
		},
		{
			name:     "Length exactly 50 valid",
			input:    strings.Repeat("a", 50),
			expected: true,
		},
		{
			name:     "Length 51 too long",
			input:    strings.Repeat("a", 51),
			expected: false,
		},
		{
			name:     "Valid all letters",
			input:    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
			expected: false, // 52 > 50 → should fail
		},
		{
			name:     "Valid all numbers",
			input:    "0123456789",
			expected: true,
		},
		{
			name:     "Valid letters and numbers",
			input:    "abc123",
			expected: true,
		},
		{
			name:     "Invalid special character",
			input:    "abc$123",
			expected: false,
		},
		{
			name:     "Invalid whitespace",
			input:    "abc 123",
			expected: false,
		},
		{
			name:     "Invalid unicode symbol",
			input:    "abc☺",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidObjectID(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func mock_database() *sql.DB {
	// Create a mock database connection for testing
	db, _, _ := sqlmock.New()
	return db
}

func TestMissingParams(t *testing.T) {

	db := mock_database()

	database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Missing student_id or course_id")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/api/stats", nil)
	c.Request = req

	// No params set
	c.Params = []gin.Param{}

	APIHandlerGetStatsForStudent(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Missing student_id")
}
func TestAPIHandlerGetStatsForStudent(t *testing.T) {
	tests := []struct {
		name            string
		studentID       string
		courseID        string
		mockFunc        func()
		expectedCode    int
		expectedMessage string
	}{
		{
			name:      "Missing Params",
			studentID: "",
			courseID:  "",
			mockFunc: func() {
				// This won't even call the DB because it fails early
			},
			expectedCode:    http.StatusBadRequest,
			expectedMessage: "Missing student_id",
		},
		{
			name:      "Invalid CourseID",
			studentID: "abc",
			courseID:  "invalidCourseID",
			mockFunc: func() {
				// No DB call either, fails on format
			},
			expectedCode:    400,
			expectedMessage: "{\"result\":\"Failed to get average grade\",\"status\":500}",
		},
		{
			name:      "DB Error",
			studentID: "abc",
			courseID:  "507f1f77bcf86cd799439011",
			mockFunc: func() {
				database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
					return 0, http.StatusInternalServerError, errors.New("DB error")
				}
			},
			expectedCode:    http.StatusInternalServerError,
			expectedMessage: "Failed to get average grade",
		},
		{
			name:      "No Grades Found",
			studentID: "abc",
			courseID:  "507f1f77bcf86cd799439011",
			mockFunc: func() {
				database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
					return 0, http.StatusNotFound, nil
				}
			},
			expectedCode:    http.StatusNotFound,
			expectedMessage: "No grades found",
		},
		{
			name:      "Success",
			studentID: "abc",
			courseID:  "507f1f77bcf86cd799439011",
			mockFunc: func() {
				database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
					return 7.5, http.StatusOK, nil
				}
			},
			expectedCode:    http.StatusOK,
			expectedMessage: "average_grade",
		},
	}

	db := mock_database()
	defer db.Close()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockFunc()

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			req, _ := http.NewRequest("GET", "/api/stats", nil)
			c.Request = req

			// Add path params if any
			c.Params = []gin.Param{
				{Key: "student_id", Value: tt.studentID},
				{Key: "course_id", Value: tt.courseID},
			}

			APIHandlerGetStatsForStudent(db, c)

			assert.Equal(t, tt.expectedCode, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedMessage)
		})
	}
}
func TestInvalidCourseID(t *testing.T) {

	db := mock_database()

	database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Missing student_id or course_id")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{
		{Key: "student_id", Value: "123"},
		{Key: "course_id", Value: "invalidObjectID"},
	}

	APIHandlerGetStatsForStudent(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

func TestDBError(t *testing.T) {

	db := mock_database()

	database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusInternalServerError, errors.New("Failed to get average grade")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{
		{Key: "student_id", Value: "123"},
		{Key: "course_id", Value: "60c72b2f9b1e8a3d4c8f9c02"}, // valid-looking ObjectID
	}

	APIHandlerGetStatsForStudent(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to get average grade")
}

func TestNoGradesFound(t *testing.T) {

	db := mock_database()

	database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusNotFound, errors.New("No grades found for this student in the course")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{
		{Key: "student_id", Value: "123"},
		{Key: "course_id", Value: "60c72b2f9b1e8a3d4c8f9c02"},
	}

	APIHandlerGetStatsForStudent(db, c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

func TestSuccessCase(t *testing.T) {

	db := mock_database()

	database.GetAvgGradeForStudent = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 92.5, http.StatusOK, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Params = []gin.Param{
		{Key: "student_id", Value: "123"},
		{Key: "course_id", Value: "60c72b2f9b1e8a3d4c8f9c02"},
	}

	APIHandlerGetStatsForStudent(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"average_grade":92.5`)
	assert.Contains(t, w.Body.String(), `"course_id":"60c72b2f9b1e8a3d4c8f9c02"`)
}

// test for APIHandlerGetStudentAverageOverTime

func TestAPIHandlerGetStudentAverageOverTime_HappyPath(t *testing.T) {
	db := mock_database()

	database.GetStudentAveragesOverTime = func(DB *sql.DB, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		// Mocked data for testing
		return []map[string]interface{}{
			{"student_id": "123", "averages": []float64{90.5, 85}, "group_by": groupBy},
		}, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/123/averages?start_date=2023-01-01&end_date=2023-01-31&group_by=week", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "student_id", Value: "123"}}

	APIHandlerGetStudentAverageOverTime(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"student_id":"123"`)
	assert.Contains(t, w.Body.String(), `"averages":[90.5,85]`)
	assert.Contains(t, w.Body.String(), `"group_by":"week"`)
}

func TestAPIHandlerGetStudentAverageOverTime_InvalidQueryParams(t *testing.T) {
	db := mock_database()

	database.GetStudentAveragesOverTime = func(DB *sql.DB, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("Invalid query parameters")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/123/averages?start_date=&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "student_id", Value: "123"}}

	APIHandlerGetStudentAverageOverTime(db, c)

	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid query parameters")
}

func TestAPIHandlerGetStudentAverageOverTime_InvalidDateFormat(t *testing.T) {
	db := mock_database()

	database.GetStudentAveragesOverTime = func(DB *sql.DB, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("Invalid date format")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/123/averages?start_date=2023-99-99&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "student_id", Value: "123"}}

	APIHandlerGetStudentAverageOverTime(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid date format")
}

func TestAPIHandlerGetStudentAverageOverTime_DatabaseError(t *testing.T) {
	db := mock_database()

	database.GetStudentAveragesOverTime = func(DB *sql.DB, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("db error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/123/averages?start_date=2023-01-01&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "student_id", Value: "123"}}

	APIHandlerGetStudentAverageOverTime(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "db error")
}

// Tests now for APIHandlerGetCourseAverageOverTime

func TestAPIHandlerGetCourseAverageOverTime_Success(t *testing.T) {
	db := mock_database()

	database.GetCourseAveragesOverTime = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return []map[string]interface{}{
			{"course_id": "abc123", "averages": []float64{75.5, 80, 82.3}, "group_by": groupBy},
		}, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/course/abc123/averages?start_date=2023-01-01&end_date=2023-01-31&group_by=week", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "course_id", Value: "abc123"}}

	APIHandlerGetCourseAverageOverTime(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"course_id":"abc123"`)
	assert.Contains(t, w.Body.String(), `"averages":[75.5,80,82.3]`)
	assert.Contains(t, w.Body.String(), `"group_by":"week"`)
}

func TestAPIHandlerGetCourseAverageOverTime_InvalidQueryParams(t *testing.T) {
	db := mock_database()

	database.GetCourseAveragesOverTime = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("Invalid query parameters")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Missing required query parameters or malformed query
	req, _ := http.NewRequest("GET", "/course/abc123/averages?start_date=&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "course_id", Value: "abc123"}}

	APIHandlerGetCourseAverageOverTime(db, c)

	assert.Equal(t, 500, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid query parameters")
}

func TestAPIHandlerGetCourseAverageOverTime_InvalidDateFormat(t *testing.T) {
	db := mock_database()

	database.GetCourseAveragesOverTime = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("Invalid date format")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/course/abc123/averages?start_date=2023-99-99&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "course_id", Value: "abc123"}}

	APIHandlerGetCourseAverageOverTime(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Invalid date format")
}

func TestAPIHandlerGetCourseAverageOverTime_DatabaseError(t *testing.T) {
	db := mock_database()

	database.GetCourseAveragesOverTime = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
		return nil, errors.New("database failure")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/course/abc123/averages?start_date=2023-01-01&end_date=2023-01-31", nil)
	c.Request = req
	c.Params = []gin.Param{{Key: "course_id", Value: "abc123"}}

	APIHandlerGetCourseAverageOverTime(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "database failure")
}

// tests for APIHandlerGetStatsForStudentTask

func TestAPIHandlerGetStatsForStudentTask_Success(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 88.5, http.StatusOK, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"average_grade":88.5`)
	assert.Contains(t, w.Body.String(), `"course_id":"507f1f77bcf86cd799439012"`)
	assert.Contains(t, w.Body.String(), `"task_id":"507f1f77bcf86cd799439013"`)
}

func TestAPIHandlerGetStatsForStudentTask_InvalidStudentID(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Invalid student_id format")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/invalidID/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "invalidID"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

func TestAPIHandlerGetStatsForStudentTask_InvalidCourseOrTaskID(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Invalid course_id or task_id format")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/invalidCourse/task/invalidTask", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "invalidCourse"},
		{Key: "task_id", Value: "invalidTask"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

func TestAPIHandlerGetStatsForStudentTask_NoGradesFound(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 0, http.StatusNotFound, errors.New("No grades found for the student in this task")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

func TestAPIHandlerGetStatsForStudentTask_DatabaseError(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 0, http.StatusInternalServerError, errors.New("database failure")
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "Failed to get average grade")
}

func TestAPIHandlerGetStatsForStudentTask_UserNotFound(t *testing.T) {
	db := mock_database()

	database.GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
		return 0, http.StatusNotFound, errors.New("No grades found for the student in this task")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)

	c.Request = req

	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetStatsForStudentTask(db, c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "{\"result\":\"Failed to get average grade\",\"status\":500}")
}

// test for APIHandlerGetStudentCourseTasksAverage

func TestAPIHandlerGetStudentCourseTasksAverage_Success(t *testing.T) {
	db := mock_database()

	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 91.5, http.StatusOK, nil
	}
	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return []map[string]interface{}{
			{"student_id": "507f1f77bcf86cd799439013", "average_grade": 88.0, "task_count": 2},
			{"student_id": "507f1f77bcf86cd799439014", "average_grade": 92.0, "task_count": 3},
		}, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"student_average":91.5`)
	assert.Contains(t, w.Body.String(), `{"course_id":"507f1f77bcf86cd799439012","other_students":[{"average_grade":88,"student_id":"507f1f77bcf86cd799439013","task_count":2},{"average_grade":92,"student_id":"507f1f77bcf86cd799439014","task_count":3}],"student_average":91.5,"student_id":"507f1f77bcf86cd799439011"}`)
}

func TestAPIHandlerGetStudentCourseTasksAverage_InvalidStudentID(t *testing.T) {
	db := mock_database()

	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Invalid student_id format")
	}

	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return nil, nil // Not really called on this case: Lucas fix
	}
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/invalid/course/507f1f77bcf86cd799439012", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "invalid"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"Invalid student_id format"`)
}
func TestAPIHandlerGetStudentCourseTasksAverage_InvalidCourseID(t *testing.T) {

	db := mock_database()

	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusBadRequest, errors.New("Invalid course_id format")
	}

	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return nil, nil // Not really called on this case: Lucas fix
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/invalid", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "invalid"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"Invalid course_id format"`)
}
func TestAPIHandlerGetStudentCourseTasksAverage_DBErrorOnStudentAvg(t *testing.T) {
	db := mock_database()

	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusInternalServerError, errors.New("db error")
	}

	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return []map[string]interface{}{
			{"student_id": "507f1f77bcf86cd799439013", "average_grade": 88.0, "task_count": 2},
			{"student_id": "507f1f77bcf86cd799439014", "average_grade": 92.0, "task_count": 3},
		}, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"db error"`)
}
func TestAPIHandlerGetStudentCourseTasksAverage_DBErrorOnOthers(t *testing.T) {

	db := mock_database()

	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 85, http.StatusOK, nil
	}

	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return nil, errors.New("db error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `{"error":"db error"}`)
}
func TestAPIHandlerGetStudentCourseTasksAverage_StudentNotFound(t *testing.T) {

	db := mock_database()
	database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
		return 0, http.StatusNotFound, errors.New("No grades found for the requested student")
	}

	database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
		return nil, nil // Not really called on this case: Lucas fix
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/student/507f1f77bcf86cd799439011/course/507f1f77bcf86cd799439012", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "student_id", Value: "507f1f77bcf86cd799439011"},
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
	}

	APIHandlerGetStudentCourseTasksAverage(db, c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), `{"error":"No grades found for the requested student"}`)
}
func TestAPIHandlerGetStudentCourseTasksAverage(t *testing.T) {
	db := mock_database()

	t.Run("Invalid student_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/bad_id/course/validcourse", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "bad$id"}, // invalid char
			{Key: "course_id", Value: "validcourse"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid student_id format")
	})

	t.Run("Invalid course_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/validstudent/course/bad_id", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "validstudent"},
			{Key: "course_id", Value: "bad#id"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid course_id format")
	})

	t.Run("GetStudentCourseTasksAverage returns error", func(t *testing.T) {
		database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
			return 0, http.StatusInternalServerError, errors.New("some DB error")
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/validstudent/course/validcourse", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "validstudent"},
			{Key: "course_id", Value: "validcourse"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "some DB error")
	})

	t.Run("GetOtherStudentsCourseAverages returns error", func(t *testing.T) {
		database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
			return 7.5, http.StatusOK, nil
		}

		database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
			return nil, errors.New("other students DB error")
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/validstudent/course/validcourse", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "validstudent"},
			{Key: "course_id", Value: "validcourse"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "other students DB error")
	})

	t.Run("No grades found for student (NotFound)", func(t *testing.T) {
		database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
			return 0, http.StatusNotFound, nil
		}

		database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"student_id": "otherstudent1", "average_grade": 6.0, "task_count": 2},
				{"student_id": "otherstudent2", "average_grade": 7.5, "task_count": 3},
			}, nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/validstudent/course/validcourse", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "validstudent"},
			{Key: "course_id", Value: "validcourse"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "No grades found")
		assert.Contains(t, w.Body.String(), "warning")
	})

	t.Run("Happy path", func(t *testing.T) {
		database.GetStudentCourseTasksAverage = func(db *sql.DB, studentID, courseID string) (float64, int, error) {
			return 8.0, http.StatusOK, nil
		}

		database.GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
			return []map[string]interface{}{
				{"student_id": "otherstudent1", "average_grade": 7.0, "task_count": 2},
				{"student_id": "otherstudent2", "average_grade": 9.0, "task_count": 3},
			}, nil
		}

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		req, _ := http.NewRequest("GET", "/api/stats/student/validstudent/course/validcourse", nil)
		c.Request = req
		c.Params = []gin.Param{
			{Key: "student_id", Value: "validstudent"},
			{Key: "course_id", Value: "validcourse"},
		}

		APIHandlerGetStudentCourseTasksAverage(db, c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "student_average")
		assert.Contains(t, w.Body.String(), "other_students")
		assert.Contains(t, w.Body.String(), "8")
	})
}

// tests for APIHandlerGetTaskAverages

func TestAPIHandlerGetTaskAverages_Success(t *testing.T) {
	db := mock_database()
	database.GetAveragesForTask = func(db *sql.DB, courseID, taskID string) ([]map[string]interface{}, error) {
		return []map[string]interface{}{
			{"average_grade": 85.0, "grade_count": 2},
			{"average_grade": 95.0, "grade_count": 3},
		}, nil
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetTaskAverages(db, c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"group_average":91`)
	assert.Contains(t, w.Body.String(), `"course_id":"507f1f77bcf86cd799439012"`)
}

func TestAPIHandlerGetTaskAverages_DBError(t *testing.T) {
	db := mock_database()
	database.GetAveragesForTask = func(db *sql.DB, courseID, taskID string) ([]map[string]interface{}, error) {
		return nil, errors.New("mock DB error")
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/course/507f1f77bcf86cd799439012/task/507f1f77bcf86cd799439013", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "course_id", Value: "507f1f77bcf86cd799439012"},
		{Key: "task_id", Value: "507f1f77bcf86cd799439013"},
	}

	APIHandlerGetTaskAverages(db, c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"mock DB error"`)
}

func TestAPIHandlerGetTaskAverages_InvalidParams(t *testing.T) {
	db := mock_database()

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req, _ := http.NewRequest("GET", "/course//task/", nil)
	c.Request = req
	c.Params = []gin.Param{
		{Key: "course_id", Value: ""},
		{Key: "task_id", Value: ""},
	}

	APIHandlerGetTaskAverages(db, c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), `"error":"Invalid course_id or task_id format"`)
}

/////////////////////////////////////////////////////////////////////////////////

func TestAPIHandlerGetCourseOnTimePercentage(t *testing.T) {
	// Save original funcs, restore after tests
	origFunc := database.GetOnTimeSubmissionPercentageForCourse
	defer func() { database.GetOnTimeSubmissionPercentageForCourse = origFunc }()

	dbMock := &sql.DB{} // dummy

	tests := []struct {
		name           string
		query          string
		courseID       string
		mockReturn     interface{}
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid request returns 200",
			query:          "start_date=2023-01-01&end_date=2023-01-31&group_by=week",
			courseID:       "course123",
			mockReturn:     map[string]interface{}{"percentage": 80},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `"course_id":"course123"`,
		},
		{
			name:           "invalid query params returns 400",
			query:          "start_date=invalid&end_date=2023-01-31",
			courseID:       "course123",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Invalid date format`,
		},
		{
			name:           "db error returns 500",
			query:          "start_date=2023-01-01&end_date=2023-01-31",
			courseID:       "course123",
			mockReturn:     nil,
			mockError:      errors.New("DB error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `DB error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database.GetOnTimeSubmissionPercentageForCourse = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
				if tt.mockError != nil {
					return nil, tt.mockError
				}
				return []map[string]interface{}{
					{"course_id": tt.courseID, "percentage": tt.mockReturn.(map[string]interface{})["percentage"]},
				}, nil
			}

			// Prepare Gin
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/courses/"+tt.courseID+"?"+tt.query, nil)
			c.Params = gin.Params{{Key: "course_id", Value: tt.courseID}}

			APIHandlerGetCourseOnTimePercentage(dbMock, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}

func TestAPIHandlerGetStudentOnTimePercentage(t *testing.T) {
	origFunc := database.GetOnTimeSubmissionPercentageForStudent
	defer func() { database.GetOnTimeSubmissionPercentageForStudent = origFunc }()

	dbMock := &sql.DB{} // dummy

	tests := []struct {
		name           string
		query          string
		courseID       string
		studentID      string
		mockReturn     interface{}
		mockError      error
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "valid request returns 200",
			query:          "start_date=2023-01-01&end_date=2023-01-31&group_by=day",
			courseID:       "course123",
			studentID:      "student456",
			mockReturn:     map[string]interface{}{"percentage": 90},
			mockError:      nil,
			expectedStatus: http.StatusOK,
			expectedBody:   `"student_id":"student456"`,
		},
		{
			name:           "invalid query params returns 400",
			query:          "start_date=bad-date&end_date=2023-01-31",
			courseID:       "course123",
			studentID:      "student456",
			mockReturn:     nil,
			mockError:      nil,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   `Invalid date format`,
		},
		{
			name:           "db error returns 500",
			query:          "start_date=2023-01-01&end_date=2023-01-31",
			courseID:       "course123",
			studentID:      "student456",
			mockReturn:     nil,
			mockError:      errors.New("DB error"),
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   `DB error`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database.GetOnTimeSubmissionPercentageForStudent = func(DB *sql.DB, courseID, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
				if tt.mockError != nil {
					return nil, tt.mockError
				}
				return []map[string]interface{}{
					{"course_id": tt.courseID, "student_id": tt.studentID, "percentage": tt.mockReturn.(map[string]interface{})["percentage"]},
				}, nil
			}

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest("GET", "/courses/"+tt.courseID+"/students/"+tt.studentID+"?"+tt.query, nil)
			c.Params = gin.Params{
				{Key: "course_id", Value: tt.courseID},
				{Key: "student_id", Value: tt.studentID},
			}

			APIHandlerGetStudentOnTimePercentage(dbMock, c)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Contains(t, w.Body.String(), tt.expectedBody)
		})
	}
}
