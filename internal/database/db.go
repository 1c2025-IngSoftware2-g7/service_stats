package database

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"service_stats/internal/model"
	"time"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB(posgresUrl string) (*sql.DB, error) {
	var err error

	DB, err = sql.Open("postgres", posgresUrl)

	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
		return nil, err
	}

	// Check if the connection is established
	if err = DB.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
		return nil, err
	}

	statement := `CREATE TABLE IF NOT EXISTS grades (
    id SERIAL PRIMARY KEY,
    student_id TEXT NOT NULL,
    course_id  TEXT NOT NULL,
    grade      NUMERIC NOT NULL,
    on_time    BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);

    CREATE TABLE IF NOT EXISTS grades_tasks (
        id SERIAL PRIMARY KEY,
        student_id TEXT NOT NULL,
        course_id  TEXT NOT NULL,
        task_id    TEXT NOT NULL,
        grade      NUMERIC NOT NULL,
        on_time    BOOLEAN NOT NULL DEFAULT TRUE,
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );
	`
	_, err = DB.Exec(statement)

	if err != nil {
		log.Fatalf("[Service Stats] error creating tables: %v", err)
		return nil, err
	}
	return DB, nil
}

var InsertGrade = func(db *sql.DB, grade model.Grade) (err error) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[Service Stats] Error starting transaction: %v", err)
		return err
	}

	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("[Service Stats] Error rolling back transaction: %v", rollbackErr)
			}
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("[Service Stats] Error committing transaction: %v", commitErr)
				err = commitErr
			}
		}
	}()

	statement := `INSERT INTO grades (student_id, course_id, grade, on_time) VALUES ($1, $2, $3, $4)`
	_, err = tx.Exec(statement, grade.StudentID, grade.CourseID, grade.Grade, grade.OnTime)
	if err != nil {
		log.Printf("[Service Stats] Error inserting grade: %v", err)
		return err
	}

	return nil
}

var GetAvgGradeForStudent = func(db *sql.DB, studentID string, courseID string) (float64, int, error) {
	tx, err := db.Begin()
	if err != nil {
		log.Printf("[Service Stats] Error starting transaction: %v", err)
		return 0, http.StatusInternalServerError, err
	}

	defer func() {
		if err == nil {
			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("[Service Stats] Error committing transaction: %v", commitErr)
				err = commitErr
			}
		} else {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				log.Printf("[Service Stats] Error rolling back transaction: %v", rollbackErr)
			}
		}
	}()

	var avgGrade float64
	statement := `SELECT AVG(grade) FROM grades WHERE student_id = $1 AND course_id = $2 GROUP BY student_id, course_id`

	err = tx.QueryRow(statement, studentID, courseID).Scan(&avgGrade)

	if err == sql.ErrNoRows {
		log.Printf("[Service Stats] No grades found for student %s in course %s", studentID, courseID)
		return 0.0, http.StatusNotFound, nil
	}

	if err != nil {
		log.Printf("[Service Stats] Error getting average grade for student %s in course %s: %v", studentID, courseID, err)
		return 0, http.StatusInternalServerError, err
	}

	return avgGrade, http.StatusOK, nil
}

// GetStudentAveragesOverTime returns student's grade averages over time
var GetStudentAveragesOverTime = func(DB *sql.DB, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var query string
	var args []interface{}

	baseQuery := `
		SELECT 
			DATE_TRUNC($1, created_at) AS period,
			AVG(grade) AS average_grade,
			COUNT(*) AS grade_count
		FROM grades
		WHERE student_id = $2
	`

	args = append(args, groupBy, studentID)
	argPos := 3

	if !startTime.IsZero() {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, startTime)
		argPos++
	}
	if !endTime.IsZero() {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, endTime)
		argPos++
	}

	query = baseQuery + " GROUP BY period ORDER BY period"

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var period time.Time
		var avgGrade float64
		var count int
		if err := rows.Scan(&period, &avgGrade, &count); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"period":        period.Format(time.RFC3339),
			"average_grade": avgGrade,
			"grade_count":   count,
		})
	}
	return results, nil
}

// GetCourseAveragesOverTime returns course's grade averages over time
var GetCourseAveragesOverTime = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var query string
	var args []interface{}

	baseQuery := `
		SELECT 
			DATE_TRUNC($1, created_at) AS period,
			AVG(grade) AS average_grade,
			COUNT(*) AS grade_count
		FROM grades
		WHERE course_id = $2
	`

	args = append(args, groupBy, courseID)
	argPos := 3

	if !startTime.IsZero() {
		baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
		args = append(args, startTime)
		argPos++
	}
	if !endTime.IsZero() {
		baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
		args = append(args, endTime)
		argPos++
	}

	query = baseQuery + " GROUP BY period ORDER BY period"

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var period time.Time
		var avgGrade float64
		var count int
		if err := rows.Scan(&period, &avgGrade, &count); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"period":        period.Format(time.RFC3339),
			"average_grade": avgGrade,
			"grade_count":   count,
		})
	}
	return results, nil
}

// InsertGradeTask inserts a new grade task
var InsertGradeTask = func(DB *sql.DB, grade model.GradeTask) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	statement := `INSERT INTO grades_tasks (student_id, course_id, task_id, grade, on_time)
				  VALUES ($1, $2, $3, $4, $5)`
	_, err = tx.Exec(statement, grade.StudentID, grade.CourseID, grade.TaskID, grade.Grade, grade.OnTime)
	if err != nil {
		log.Printf("[Service Stats] Error inserting grade task: %v", err)
		return err
	}

	return tx.Commit()
}

// GetAvgGradeTaskForStudent returns student's average in one task
var GetAvgGradeTaskForStudent = func(DB *sql.DB, studentID string, courseID string, taskID string) (float64, int, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var avgGrade float64
	statement := `SELECT AVG(grade) FROM grades_tasks
				  WHERE student_id = $1 AND course_id = $2 AND task_id = $3
				  GROUP BY student_id, course_id, task_id`

	err = tx.QueryRow(statement, studentID, courseID, taskID).Scan(&avgGrade)
	if err == sql.ErrNoRows {
		log.Printf("[Service Stats] No grades found for student %s in course %s task %s", studentID, courseID, taskID)
		return 0.0, http.StatusNotFound, nil
	}
	if err != nil {
		log.Printf("[Service Stats] Error getting average grade for task: %v", err)
		return 0, http.StatusInternalServerError, err
	}

	return avgGrade, http.StatusOK, nil
}

// GetStudentCourseTasksAverage returns average for student in all course tasks
var GetStudentCourseTasksAverage = func(DB *sql.DB, studentID string, courseID string) (float64, int, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}
	defer tx.Rollback()

	var avgGrade float64
	statement := `SELECT AVG(grade) FROM grades_tasks WHERE student_id = $1 AND course_id = $2`

	err = tx.QueryRow(statement, studentID, courseID).Scan(&avgGrade)
	if err == sql.ErrNoRows {
		return 0.0, http.StatusNotFound, nil
	}
	if err != nil {
		return 0, http.StatusInternalServerError, err
	}

	return avgGrade, http.StatusOK, nil
}

// GetOtherStudentsCourseAverages returns averages for all other students in a course
var GetOtherStudentsCourseAverages = func(DB *sql.DB, studentID string, courseID string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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

	rows, err := tx.Query(query, courseID, studentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var sID string
		var avgGrade float64
		var taskCount int
		if err := rows.Scan(&sID, &avgGrade, &taskCount); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"student_id":    sID,
			"average_grade": avgGrade,
			"task_count":    taskCount,
		})
	}

	return results, nil
}

// GetAveragesForTask returns averages for all students in a task
var GetAveragesForTask = func(DB *sql.DB, courseID string, taskID string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

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

	rows, err := tx.Query(query, courseID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var sID string
		var avgGrade float64
		var gradeCount int
		if err := rows.Scan(&sID, &avgGrade, &gradeCount); err != nil {
			return nil, err
		}
		results = append(results, map[string]interface{}{
			"student_id":    sID,
			"average_grade": avgGrade,
			"grade_count":   gradeCount,
		})
	}

	return results, nil
}

var GetOnTimeSubmissionPercentageForCourse = func(DB *sql.DB, courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback() // rollback is safe even if already committed
	}()

	var query string
	var args []interface{}

	if groupBy == "" {
		baseQuery := `
			SELECT
				'all_time' AS period,
				COUNT(*) FILTER (WHERE on_time = true) AS on_time_count,
				COUNT(*) AS total_count,
				COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage
			FROM grades_tasks
			WHERE course_id = $1
		`
		args = append(args, courseID)
		argPos := 2

		if !startTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
			args = append(args, startTime)
			argPos++
		}

		if !endTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
			args = append(args, endTime)
		}

		query = baseQuery
	} else {
		baseQuery := `
			SELECT
				DATE_TRUNC($1, created_at) AS period,
				COUNT(*) FILTER (WHERE on_time = true) AS on_time_count,
				COUNT(*) AS total_count,
				COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage
			FROM grades_tasks
			WHERE course_id = $2
		`

		args = append(args, groupBy, courseID)
		argPos := 3

		if !startTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
			args = append(args, startTime)
			argPos++
		}

		if !endTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
			args = append(args, endTime)
		}

		query = baseQuery + " GROUP BY period ORDER BY period"
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var period interface{}
		var onTimeCount, totalCount int
		var percentage float64

		if err := rows.Scan(&period, &onTimeCount, &totalCount, &percentage); err != nil {
			return nil, err
		}

		if t, ok := period.(time.Time); ok {
			period = t.Format(time.RFC3339)
		}

		results = append(results, map[string]interface{}{
			"period":        period,
			"on_time_count": onTimeCount,
			"total_count":   totalCount,
			"percentage":    percentage,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}

// GetOnTimeSubmissionPercentageForStudent devuelve el porcentaje de tareas entregadas a tiempo para un estudiante en un curso
var GetOnTimeSubmissionPercentageForStudent = func(DB *sql.DB, courseID, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
	tx, err := DB.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = tx.Rollback() // Safe rollback in case of early return or failure
	}()

	var query string
	var args []interface{}

	if groupBy == "" {
		baseQuery := `
			SELECT
				'all_time' AS period,
				COUNT(*) FILTER (WHERE on_time = true) AS on_time_count,
				COUNT(*) AS total_count,
				COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage
			FROM grades_tasks
			WHERE course_id = $1 AND student_id = $2
		`
		args = append(args, courseID, studentID)
		argPos := 3

		if !startTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
			args = append(args, startTime)
			argPos++
		}

		if !endTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
			args = append(args, endTime)
		}

		query = baseQuery
	} else {
		baseQuery := `
			SELECT
				DATE_TRUNC($1, created_at) AS period,
				COUNT(*) FILTER (WHERE on_time = true) AS on_time_count,
				COUNT(*) AS total_count,
				COALESCE((COUNT(*) FILTER (WHERE on_time = true) * 100.0 / NULLIF(COUNT(*), 0)), 0) AS percentage
			FROM grades_tasks
			WHERE course_id = $2 AND student_id = $3
		`

		args = append(args, groupBy, courseID, studentID)
		argPos := 4

		if !startTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at >= $%d", argPos)
			args = append(args, startTime)
			argPos++
		}

		if !endTime.IsZero() {
			baseQuery += fmt.Sprintf(" AND created_at <= $%d", argPos)
			args = append(args, endTime)
		}

		query = baseQuery + " GROUP BY period ORDER BY period"
	}

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var period interface{}
		var onTimeCount, totalCount int
		var percentage float64

		if err := rows.Scan(&period, &onTimeCount, &totalCount, &percentage); err != nil {
			return nil, err
		}

		if t, ok := period.(time.Time); ok {
			period = t.Format(time.RFC3339)
		}

		results = append(results, map[string]interface{}{
			"period":        period,
			"on_time_count": onTimeCount,
			"total_count":   totalCount,
			"percentage":    percentage,
		})
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return results, nil
}
