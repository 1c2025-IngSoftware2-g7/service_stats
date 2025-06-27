package database

import (
	"database/sql"
	"fmt"
	"service_stats/internal/model"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
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
