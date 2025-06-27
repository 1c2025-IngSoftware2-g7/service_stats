package database

import (
	"database/sql"
	"errors"
	"fmt"
	"service_stats/internal/model"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("error opening a stub database connection: %s", err)
	}
	return db, mock
}

func TestInitDB(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	// Expect a transaction to begin
	mock.ExpectBegin()

	// Correct regex matching your actual SQL query with $1, $2 and GROUP BY
	query := `SELECT AVG(grade) FROM grades WHERE student_id = $1 AND course_id = $2 GROUP BY student_id, course_id`

	rows := sqlmock.NewRows([]string{"avg_grade"}).AddRow(85.0)
	mock.ExpectQuery(query).
		WithArgs("student1", "course1").
		WillReturnRows(rows)

	// Expect the transaction to commit
	mock.ExpectCommit()

	avgGrade, code, err := GetAvgGradeForStudent(db, "student1", "course1")
	if err != nil {
		t.Errorf("error was not expected while getting average grade: %s", err)
	}

	if code != 200 {
		t.Errorf("expected status code 200, got %d", code)
	}
	if avgGrade != 85.0 {
		t.Errorf("expected average grade 85.0, got %f", avgGrade)
	}
}

func TestInsertGrade(t *testing.T) {
	db, mock, err := sqlmock.New()

	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO grades`).WithArgs("student1", "course1", 95.0, true).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	grade := model.Grade{
		StudentID: "student1",
		CourseID:  "course1",
		Grade:     95.0,
		OnTime:    true,
	}

	if err := InsertGrade(db, grade); err != nil {
		t.Errorf("error was not expected while inserting grade: %s", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}

func TestGetAvgGradeForStudent(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("unexpected error when opening a stub database connection: %s", err)
	}
	defer db.Close()

	// Expect a transaction to begin
	mock.ExpectBegin()

	// Correct regex matching your actual SQL query with $1, $2 and GROUP BY
	query := `SELECT AVG\(grade\) FROM grades WHERE student_id = \$1 AND course_id = \$2 GROUP BY student_id, course_id`

	rows := sqlmock.NewRows([]string{"avg_grade"}).AddRow(85.0)
	mock.ExpectQuery(query).
		WithArgs("student1", "course1").
		WillReturnRows(rows)

	// Expect the transaction to commit
	mock.ExpectCommit()

	avgGrade, code, err := GetAvgGradeForStudent(db, "student1", "course1")
	if err != nil {
		t.Errorf("error was not expected while getting average grade: %s", err)
	}

	if code != 200 {
		t.Errorf("expected status code 200, got %d", code)
	}
	if avgGrade != 85.0 {
		t.Errorf("expected average grade 85.0, got %f", avgGrade)
	}
}

func TestGetCourseAveragesOverTime(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	groupBy := "day"
	courseID := "course123"
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()

	// Make sure SQL string exactly matches your production SQL including whitespace.
	query := `SELECT 
			DATE_TRUNC($1, created_at) AS period,
			AVG(grade) AS average_grade,
			COUNT(*) AS grade_count
		FROM grades
		WHERE course_id = $2 AND created_at >= $3 AND created_at <= $4 GROUP BY period ORDER BY period`

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"period", "average_grade", "grade_count"}).
		AddRow(start, 8.5, 10)
	mock.ExpectQuery(query).
		WithArgs(groupBy, courseID, start, end).
		WillReturnRows(rows)
	mock.ExpectRollback()

	results, err := GetCourseAveragesOverTime(db, courseID, start, end, groupBy)
	assert.NoError(t, err)
	assert.Len(t, results, 1)
	assert.Equal(t, 8.5, results[0]["average_grade"])
}

func TestInsertGradeTask(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	grade := model.GradeTask{
		StudentID: "stu1",
		CourseID:  "c1",
		TaskID:    "t1",
		Grade:     9.0,
		OnTime:    true,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades_tasks (student_id, course_id, task_id, grade, on_time) VALUES ($1, $2, $3, $4, $5)").
		WithArgs(grade.StudentID, grade.CourseID, grade.TaskID, grade.Grade, grade.OnTime).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err := InsertGradeTask(db, grade)
	assert.NoError(t, err)
}

func TestGetAvgGradeTaskForStudent(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT AVG(grade) FROM grades_tasks
				  WHERE student_id = $1 AND course_id = $2 AND task_id = $3
				  GROUP BY student_id, course_id, task_id`).
		WithArgs("stu1", "c1", "t1").
		WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(8.0))
	mock.ExpectRollback()

	avg, code, err := GetAvgGradeTaskForStudent(db, "stu1", "c1", "t1")
	assert.NoError(t, err)
	assert.Equal(t, 8.0, avg)
	assert.Equal(t, 200, code)
}

func TestGetStudentCourseTasksAverage(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT AVG(grade) FROM grades_tasks WHERE student_id = $1 AND course_id = $2").
		WithArgs("stu1", "c1").
		WillReturnRows(sqlmock.NewRows([]string{"avg"}).AddRow(7.5))
	mock.ExpectRollback()

	avg, code, err := GetStudentCourseTasksAverage(db, "stu1", "c1")
	assert.NoError(t, err)
	assert.Equal(t, 7.5, avg)
	assert.Equal(t, 200, code)
}

func TestGetOtherStudentsCourseAverages(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	query := `
		SELECT
			student_id,
			AVG(grade) as average_grade,
			COUNT(*) as task_count
		FROM grades_tasks
		WHERE course_id = $1 AND student_id != $2
		GROUP BY student_id
		ORDER BY average_grade DESC
	`
	mock.ExpectBegin()
	mock.ExpectQuery(query).
		WithArgs("c1", "stu1").
		WillReturnRows(sqlmock.NewRows([]string{"student_id", "average_grade", "task_count"}).
			AddRow("stu2", 6.0, 2))
	mock.ExpectRollback()

	res, err := GetOtherStudentsCourseAverages(db, "stu1", "c1")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "stu2", res[0]["student_id"])
	assert.Equal(t, 6.0, res[0]["average_grade"])
	assert.Equal(t, 2, res[0]["task_count"])
}

func TestGetAveragesForTask(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	query := `
	SELECT
		student_id,
		AVG(grade) as average_grade,
		COUNT(*) as grade_count
	FROM grades_tasks
	WHERE course_id = $1 AND task_id = $2
	GROUP BY student_id
	ORDER BY average_grade DESC
	`
	mock.ExpectBegin()
	mock.ExpectQuery(query).
		WithArgs("c1", "t1").
		WillReturnRows(sqlmock.NewRows([]string{"student_id", "average_grade", "grade_count"}).
			AddRow("stu1", 9.5, 1))
	mock.ExpectRollback()

	res, err := GetAveragesForTask(db, "c1", "t1")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "stu1", res[0]["student_id"])
	assert.Equal(t, 9.5, res[0]["average_grade"])
	assert.Equal(t, 1, res[0]["grade_count"])
}

func TestGetOnTimeSubmissionPercentageForCourse(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expected query string (partial match)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 'all_time' AS period").
		WithArgs("course1").
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow("all_time", 8, 10, 80.0))
	mock.ExpectCommit()

	res, err := GetOnTimeSubmissionPercentageForCourse(db, "course1", time.Time{}, time.Time{}, "")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "all_time", res[0]["period"])
	assert.Equal(t, int(8), res[0]["on_time_count"])
	assert.Equal(t, int(10), res[0]["total_count"])
	assert.Equal(t, 80.0, res[0]["percentage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOnTimeSubmissionPercentageForStudent(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	// Expected query string (partial match)
	mock.ExpectBegin()
	mock.ExpectQuery("SELECT\\s+'all_time' AS period").
		WithArgs("course1", "student1").
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow("all_time", 4, 5, 80.0))
	mock.ExpectCommit()

	res, err := GetOnTimeSubmissionPercentageForStudent(db, "course1", "student1", time.Time{}, time.Time{}, "")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "all_time", res[0]["period"])
	assert.Equal(t, int(4), res[0]["on_time_count"])
	assert.Equal(t, int(5), res[0]["total_count"])
	assert.Equal(t, 80.0, res[0]["percentage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertGrade_DBError(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	grade := model.Grade{
		StudentID: "student1",
		CourseID:  "course1",
		Grade:     95.0,
		OnTime:    true,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades (student_id, course_id, grade, on_time) VALUES ($1, $2, $3, $4)").
		WithArgs(grade.StudentID, grade.CourseID, grade.Grade, grade.OnTime).
		WillReturnError(fmt.Errorf("db error"))
	mock.ExpectRollback()

	err := InsertGrade(db, grade)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestGetAvgGradeForStudent_NoRows(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	query := `SELECT AVG(grade) FROM grades WHERE student_id = $1 AND course_id = $2 GROUP BY student_id, course_id`
	mock.ExpectQuery(query).
		WithArgs("studentX", "courseX").
		WillReturnRows(sqlmock.NewRows([]string{"avg_grade"})) // no rows
	mock.ExpectCommit()

	grade, code, err := GetAvgGradeForStudent(db, "studentX", "courseX")
	assert.Equal(t, 404, code)
	assert.Equal(t, 0.0, grade)
	assert.NoError(t, err)
}

func TestInsertGradeTask_Failure(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	grade := model.GradeTask{
		StudentID: "stu1",
		CourseID:  "c1",
		TaskID:    "t1",
		Grade:     9.0,
		OnTime:    true,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades_tasks (student_id, course_id, task_id, grade, on_time) VALUES ($1, $2, $3, $4, $5)").
		WithArgs(grade.StudentID, grade.CourseID, grade.TaskID, grade.Grade, grade.OnTime).
		WillReturnError(fmt.Errorf("db error"))
	mock.ExpectRollback()

	err := InsertGradeTask(db, grade)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestGetAvgGradeTaskForStudent_DBError(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT AVG(grade) FROM grades_tasks
				  WHERE student_id = $1 AND course_id = $2 AND task_id = $3
				  GROUP BY student_id, course_id, task_id`).
		WithArgs("stu1", "c1", "t1").
		WillReturnError(fmt.Errorf("query failed"))
	mock.ExpectRollback()

	avg, code, err := GetAvgGradeTaskForStudent(db, "stu1", "c1", "t1")
	assert.Error(t, err)
	assert.Equal(t, 500, code)
	assert.Equal(t, 0.0, avg)
}

func TestGetOtherStudentsCourseAverages_Empty(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	query := `
		SELECT
			student_id,
			AVG(grade) as average_grade,
			COUNT(*) as task_count
		FROM grades_tasks
		WHERE course_id = $1 AND student_id != $2
		GROUP BY student_id
		ORDER BY average_grade DESC
	`
	mock.ExpectBegin()
	mock.ExpectQuery(query).
		WithArgs("c1", "stu1").
		WillReturnRows(sqlmock.NewRows([]string{"student_id", "average_grade", "task_count"}))
	mock.ExpectRollback()

	res, err := GetOtherStudentsCourseAverages(db, "stu1", "c1")
	assert.NoError(t, err)
	assert.Len(t, res, 0)
}

func TestGetOnTimeSubmissionPercentageForCourse_ZeroTotal(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 'all_time' AS period, COUNT(*) FILTER (WHERE on_time = true) AS on_time_count, COUNT(*) AS total_count, COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage FROM grades_tasks WHERE course_id = $1").
		WithArgs("course1").
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow("all_time", 0, 0, 0.0))
	mock.ExpectCommit()

	res, err := GetOnTimeSubmissionPercentageForCourse(db, "course1", time.Time{}, time.Time{}, "")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, 0.0, res[0]["percentage"])
}

func TestGetCourseAveragesOverTime_MultipleRows(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	groupBy := "day"
	courseID := "course123"
	start := time.Now().Add(-7 * 24 * time.Hour)
	end := time.Now()

	query := `SELECT 
			DATE_TRUNC($1, created_at) AS period,
			AVG(grade) AS average_grade,
			COUNT(*) AS grade_count
		FROM grades
		WHERE course_id = $2 AND created_at >= $3 AND created_at <= $4 GROUP BY period ORDER BY period`

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"period", "average_grade", "grade_count"}).
		AddRow(start.Add(24*time.Hour), 7.5, 5).
		AddRow(start.Add(48*time.Hour), 8.0, 8)
	mock.ExpectQuery(query).
		WithArgs(groupBy, courseID, start, end).
		WillReturnRows(rows)
	mock.ExpectRollback()

	results, err := GetCourseAveragesOverTime(db, courseID, start, end, groupBy)
	assert.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, 7.5, results[0]["average_grade"])
	assert.Equal(t, 8.0, results[1]["average_grade"])
}

func TestGetOnTimeSubmissionPercentageForCourse_NoResults(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 'all_time' AS period, COUNT(*) FILTER (WHERE on_time = true) AS on_time_count, COUNT(*) AS total_count, COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage FROM grades_tasks WHERE course_id = $1").
		WithArgs("course1").
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"})) // no rows
	mock.ExpectCommit()

	results, err := GetOnTimeSubmissionPercentageForCourse(db, "course1", time.Time{}, time.Time{}, "")
	assert.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestGetOnTimeSubmissionPercentageForStudent_DBError(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT\\s+'all_time' AS period").
		WithArgs("course1", "student1").
		WillReturnError(fmt.Errorf("db error"))
	mock.ExpectRollback()

	results, err := GetOnTimeSubmissionPercentageForStudent(db, "course1", "student1", time.Time{}, time.Time{}, "")
	assert.Error(t, err)
	assert.Nil(t, results)
}

func TestInsertGradeTask_DBError(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	grade := model.GradeTask{
		StudentID: "stu1",
		CourseID:  "c1",
		TaskID:    "t1",
		Grade:     9.0,
		OnTime:    true,
	}

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades_tasks (student_id, course_id, task_id, grade, on_time) VALUES ($1, $2, $3, $4, $5)").
		WithArgs(grade.StudentID, grade.CourseID, grade.TaskID, grade.Grade, grade.OnTime).
		WillReturnError(fmt.Errorf("db error"))
	mock.ExpectRollback()

	err := InsertGradeTask(db, grade)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestGetOtherStudentsCourseAverages_EmptyResult(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	query := `
		SELECT
			student_id,
			AVG(grade) as average_grade,
			COUNT(*) as task_count
		FROM grades_tasks
		WHERE course_id = $1 AND student_id != $2
		GROUP BY student_id
		ORDER BY average_grade DESC
	`
	mock.ExpectBegin()
	mock.ExpectQuery(query).
		WithArgs("c1", "stu1").
		WillReturnRows(sqlmock.NewRows([]string{"student_id", "average_grade", "task_count"})) // no rows
	mock.ExpectRollback()

	res, err := GetOtherStudentsCourseAverages(db, "stu1", "c1")
	assert.NoError(t, err)
	assert.Len(t, res, 0)
}

func TestGetAveragesForTask_DBError(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	query := `
	SELECT
		student_id,
		AVG(grade) as average_grade,
		COUNT(*) as grade_count
	FROM grades_tasks
	WHERE course_id = $1 AND task_id = $2
	GROUP BY student_id
	ORDER BY average_grade DESC
	`
	mock.ExpectBegin()
	mock.ExpectQuery(query).
		WithArgs("c1", "t1").
		WillReturnError(fmt.Errorf("db error"))
	mock.ExpectRollback()

	res, err := GetAveragesForTask(db, "c1", "t1")
	assert.Error(t, err)
	assert.Nil(t, res)
}

func TestGetStudentAveragesOverTime(t *testing.T) {
	db, mock := setupDB(t)
	defer db.Close()

	mock.ExpectBegin()

	groupBy := "day"
	studentID := "student1"
	start := time.Date(2025, 6, 20, 0, 0, 0, 0, time.UTC)
	end := time.Date(2025, 6, 27, 0, 0, 0, 0, time.UTC)

	// Expect the query that will be executed inside the function
	query := `SELECT DATE_TRUNC($1, created_at) AS period, AVG(grade) AS average_grade, COUNT(*) AS grade_count FROM grades WHERE student_id = $2 AND created_at >= $3 AND created_at <= $4 GROUP BY period ORDER BY period`

	// Simulated rows returned from DB
	rows := sqlmock.NewRows([]string{"period", "average_grade", "grade_count"}).
		AddRow(start, 87.5, 2).
		AddRow(end, 90.0, 1)

	mock.ExpectQuery(query).
		WithArgs(groupBy, studentID, start, end).
		WillReturnRows(rows)

	mock.ExpectRollback()

	results, err := GetStudentAveragesOverTime(db, studentID, start, end, groupBy)
	require.NoError(t, err)
	require.Len(t, results, 2)

	assert.Equal(t, "2025-06-20T00:00:00Z", results[0]["period"])
	assert.Equal(t, 87.5, results[0]["average_grade"])
	assert.Equal(t, int(2), results[0]["grade_count"])

	assert.Equal(t, "2025-06-27T00:00:00Z", results[1]["period"])
	assert.Equal(t, 90.0, results[1]["average_grade"])
	assert.Equal(t, int(1), results[1]["grade_count"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertGrade_RollbackOnExecError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades").
		WithArgs("student1", "course1", 90.0, true).
		WillReturnError(errors.New("insert failed"))
	mock.ExpectRollback()

	grade := model.Grade{
		StudentID: "student1",
		CourseID:  "course1",
		Grade:     90.0,
		OnTime:    true,
	}

	err = InsertGrade(db, grade)
	assert.Error(t, err)
	assert.EqualError(t, err, "insert failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertGrade_CommitError(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO grades").
		WithArgs("student1", "course1", 90.0, true).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit().WillReturnError(errors.New("commit failed"))

	grade := model.Grade{
		StudentID: "student1",
		CourseID:  "course1",
		Grade:     90.0,
		OnTime:    true,
	}

	err = InsertGrade(db, grade)
	assert.Error(t, err)
	assert.EqualError(t, err, "commit failed")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOnTimeSubmissionPercentageForCourse_WithStartAndEndTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)

	// Expected SQL pattern — note we escape parenthesis and ignore spacing
	mock.ExpectBegin()

	mock.ExpectQuery(`SELECT\s+'all_time'\s+AS\s+period,\s+COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+AS\s+on_time_count,\s+COUNT\(\*\)\s+AS\s+total_count,\s+COALESCE\(\(COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+\*\s+100\.0\s+/\s+NULLIF\(COUNT\(\*\),\s+0\)\),\s+0\)\s+AS\s+percentage\s+FROM\s+grades_tasks\s+WHERE\s+course_id\s+=\s+\$\d+\s+AND\s+created_at\s+>=\s+\$\d+\s+AND\s+created_at\s+<=\s+\$\d+`).
		WithArgs("course123", startTime, endTime).
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow("2023-01-01T00:00:00Z", 10, 20, 50.0),
		)

	mock.ExpectCommit()

	results, err := GetOnTimeSubmissionPercentageForCourse(db, "course123", startTime, endTime, "")
	assert.NoError(t, err)
	assert.Len(t, results, 1)

	result := results[0]
	assert.Equal(t, "2023-01-01T00:00:00Z", result["period"])
	assert.Equal(t, int(10), result["on_time_count"])
	assert.Equal(t, int(20), result["total_count"])
	assert.Equal(t, 50.0, result["percentage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOnTimeSubmissionPercentageForCourse_WithGroupByStartEndTime(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	groupBy := "day"
	courseID := "course123"
	startTime := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)

	mock.ExpectBegin()

	// Pattern match the query using regex (be lenient on whitespace)
	mock.ExpectQuery(`SELECT\s+DATE_TRUNC\(\$1,\s+created_at\)\s+AS\s+period,\s+COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+AS\s+on_time_count,\s+COUNT\(\*\)\s+AS\s+total_count,\s+COALESCE\(\(COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+\*\s+100\.0\s+/\s+NULLIF\(COUNT\(\*\),\s+0\)\),\s+0\)\s+AS\s+percentage\s+FROM\s+grades_tasks\s+WHERE\s+course_id\s+=\s+\$\d+\s+AND\s+created_at\s+>=\s+\$\d+\s+AND\s+created_at\s+<=\s+\$\d+\s+GROUP\s+BY\s+period\s+ORDER\s+BY\s+period`).
		WithArgs(groupBy, courseID, startTime, endTime).
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow(time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC), 5, 10, 50.0).
			AddRow(time.Date(2023, 6, 20, 0, 0, 0, 0, time.UTC), 7, 14, 50.0),
		)

	mock.ExpectCommit()

	results, err := GetOnTimeSubmissionPercentageForCourse(db, courseID, startTime, endTime, groupBy)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	assert.Equal(t, "2023-06-10T00:00:00Z", results[0]["period"])
	assert.Equal(t, int(5), results[0]["on_time_count"])
	assert.Equal(t, int(10), results[0]["total_count"])
	assert.Equal(t, 50.0, results[0]["percentage"])

	assert.Equal(t, "2023-06-20T00:00:00Z", results[1]["period"])
	assert.Equal(t, int(7), results[1]["on_time_count"])
	assert.Equal(t, int(14), results[1]["total_count"])
	assert.Equal(t, 50.0, results[1]["percentage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetOnTimeSubmissionPercentageForStudent_WithGroupByOnly(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	courseID := "course456"
	studentID := "student789"
	groupBy := "week"
	startTime := time.Time{} // zero
	endTime := time.Time{}   // zero

	mock.ExpectBegin()

	// We expect 3 parameters: groupBy, courseID, studentID
	mock.ExpectQuery(`SELECT\s+DATE_TRUNC\(\$1,\s*created_at\)\s+AS\s+period,\s+COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+AS\s+on_time_count,\s+COUNT\(\*\)\s+AS\s+total_count,\s+COALESCE\(\(COUNT\(\*\)\s+FILTER\s+\(WHERE\s+on_time\s+=\s+true\)\s+\*\s+100\.0\s*/\s*NULLIF\(COUNT\(\*\),\s*0\)\),\s*0\)\s+AS\s+percentage\s+FROM\s+grades_tasks\s+WHERE\s+course_id\s+=\s+\$\d+\s+AND\s+student_id\s+=\s+\$\d+\s+GROUP\s+BY\s+period\s+ORDER\s+BY\s+period`).
		WithArgs(groupBy, courseID, studentID).
		WillReturnRows(sqlmock.NewRows([]string{"period", "on_time_count", "total_count", "percentage"}).
			AddRow(time.Date(2023, 6, 3, 0, 0, 0, 0, time.UTC), 8, 10, 80.0).
			AddRow(time.Date(2023, 6, 10, 0, 0, 0, 0, time.UTC), 9, 12, 75.0),
		)

	mock.ExpectCommit()

	results, err := GetOnTimeSubmissionPercentageForStudent(db, courseID, studentID, startTime, endTime, groupBy)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	assert.Equal(t, "2023-06-03T00:00:00Z", results[0]["period"])
	assert.Equal(t, int(8), results[0]["on_time_count"])
	assert.Equal(t, int(10), results[0]["total_count"])
	assert.Equal(t, 80.0, results[0]["percentage"])

	assert.Equal(t, "2023-06-10T00:00:00Z", results[1]["period"])
	assert.Equal(t, int(9), results[1]["on_time_count"])
	assert.Equal(t, int(12), results[1]["total_count"])
	assert.Equal(t, 75.0, results[1]["percentage"])

	assert.NoError(t, mock.ExpectationsWereMet())
}
