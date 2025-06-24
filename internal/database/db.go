package database

import (
	"database/sql"
	"log"
	"net/http"
	"service_stats/internal/model"

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
    student_id UUID NOT NULL,
    course_id  UUID NOT NULL,
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