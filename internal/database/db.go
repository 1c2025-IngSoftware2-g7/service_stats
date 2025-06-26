package database

import (
	"database/sql"
	"log"
	"net/http"
	"service_stats/internal/model"
	"time"
	"fmt"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// InitDB initializes the database connection
func InitDB(posgresUrl string) error {
	var err error

	DB, err = sql.Open("postgres", posgresUrl)

	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	// Check if the connection is established
	if err = DB.Ping(); err != nil {
		log.Fatalf("Error pinging the database: %v", err)
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
	}
	return err
}

func InsertGrade(grade model.Grade) error {
	statement := `INSERT INTO grades (student_id, course_id, grade, on_time) VALUES ($1, $2, $3, $4)`
	_, err := DB.Exec(statement, grade.StudentID, grade.CourseID, grade.Grade, grade.OnTime)

	if err != nil {
		log.Printf("[Service Stats] Error inserting grade: %v", err)
		return err
	}

	return nil
}

func GetAvgGradeForStudent(studentID string, courseID string) (float64, error, int) {
	var avgGrade float64
	statement := `SELECT AVG(grade) FROM grades WHERE student_id = $1 AND course_id = $2 GROUP BY student_id, course_id`
	err := DB.QueryRow(statement, studentID, courseID).Scan(&avgGrade)

	// in case there are no grades for the student, we return 0
	if err == sql.ErrNoRows {
		log.Printf("[Service Stats] No grades found for student %s in course %s", studentID, courseID)
		return 0.0, nil, http.StatusNotFound
	}

	// If there is an error other than no rows, log it and return the error
	if err != nil {
		log.Printf("[Service Stats] Error getting average grade for student %s: %v", studentID, err)
		return 0, err, http.StatusInternalServerError
	}

	// Best case scenario, we return the average grade
	return avgGrade, nil, http.StatusOK
}

// Función para obtener promedios de estudiante por período
func GetStudentAveragesOverTime(studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, args...)
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

// Función para obtener promedios de cursos por período
func GetCourseAveragesOverTime(courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, args...)
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

// InsertGradeTask inserts a new grade task into the database
func InsertGradeTask(grade model.GradeTask) error {
    statement := `INSERT INTO grades_tasks (student_id, course_id, task_id, grade, on_time)
                  VALUES ($1, $2, $3, $4, $5)`
    _, err := DB.Exec(statement, grade.StudentID, grade.CourseID, grade.TaskID, grade.Grade, grade.OnTime)

    if err != nil {
        log.Printf("[Service Stats] Error inserting grade task: %v", err)
        return err
    }

    return nil
}

// GetAvgGradeTaskForStudent returns the average grade for a student in a specific task
func GetAvgGradeTaskForStudent(studentID string, courseID string, taskID string) (float64, error, int) {
    var avgGrade float64
    statement := `SELECT AVG(grade) FROM grades_tasks
                  WHERE student_id = $1 AND course_id = $2 AND task_id = $3
                  GROUP BY student_id, course_id, task_id`

    err := DB.QueryRow(statement, studentID, courseID, taskID).Scan(&avgGrade)

    if err == sql.ErrNoRows {
        log.Printf("[Service Stats] No grades found for student %s in course %s task %s",
                   studentID, courseID, taskID)
        return 0.0, nil, http.StatusNotFound
    }

    if err != nil {
        log.Printf("[Service Stats] Error getting average grade for task: %v", err)
        return 0, err, http.StatusInternalServerError
    }

    return avgGrade, nil, http.StatusOK
}

// GetStudentCourseTasksAverage devuelve el promedio de un estudiante en todas las tasks de un curso
func GetStudentCourseTasksAverage(studentID string, courseID string) (float64, error, int) {
    var avgGrade float64
    statement := `SELECT AVG(grade) FROM grades_tasks
                  WHERE student_id = $1 AND course_id = $2`

    err := DB.QueryRow(statement, studentID, courseID).Scan(&avgGrade)

    if err == sql.ErrNoRows {
        return 0.0, nil, http.StatusNotFound
    }
    if err != nil {
        return 0, err, http.StatusInternalServerError
    }

    return avgGrade, nil, http.StatusOK
}

// GetOtherStudentsCourseAverages devuelve los promedios de otros estudiantes en el curso
func GetOtherStudentsCourseAverages(studentID string, courseID string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, courseID, studentID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []map[string]interface{}
    for rows.Next() {
        var studentID string
        var avgGrade float64
        var taskCount int

        if err := rows.Scan(&studentID, &avgGrade, &taskCount); err != nil {
            return nil, err
        }

        results = append(results, map[string]interface{}{
            "student_id":    studentID,
            "average_grade": avgGrade,
            "task_count":   taskCount,
        })
    }

    return results, nil
}

// GetAveragesForTask devuelve los promedios de todos los estudiantes para una task específica
func GetAveragesForTask(courseID string, taskID string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, courseID, taskID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()

    var results []map[string]interface{}
    for rows.Next() {
        var studentID string
        var avgGrade float64
        var gradeCount int

        if err := rows.Scan(&studentID, &avgGrade, &gradeCount); err != nil {
            return nil, err
        }

        results = append(results, map[string]interface{}{
            "student_id":    studentID,
            "average_grade": avgGrade,
            "grade_count":   gradeCount,
        })
    }

    return results, nil
}

func GetOnTimeSubmissionPercentageForCourse(courseID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, args...)
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
            "percentage":   percentage,
        })
    }

    return results, nil
}

// GetOnTimeSubmissionPercentageForStudent devuelve el porcentaje de tareas entregadas a tiempo para un estudiante en un curso
func GetOnTimeSubmissionPercentageForStudent(courseID, studentID string, startTime, endTime time.Time, groupBy string) ([]map[string]interface{}, error) {
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

    rows, err := DB.Query(query, args...)
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

    return results, nil
}