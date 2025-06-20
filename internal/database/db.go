package database

import (
	"database/sql"
	"log"
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
    on_time    BOOLEAN NOT NULL DEFAULT TRUE
	);
	`
	_, err = DB.Exec(statement)

	if err != nil {
		log.Fatalf("Error creating tables: %v", err)
	}
	return err
}

func InsertGrade(grade model.Grade) error {
	statement := `INSERT INTO grades (student_id, course_id, grade, on_time) VALUES ($1, $2, $3, $4)`
	_, err := DB.Exec(statement, grade.StudentID, grade.CourseID, grade.Grade, grade.OnTime)

	if err != nil {
		log.Printf("Error inserting grade: %v", err)
		return err
	}

	return nil
}

func GetAvgGradeForStudent(studentID string, courseID string) (float64, error) {
	var avgGrade float64
	statement := `SELECT AVG(grade) FROM grades WHERE student_id = $1 AND course_id = $2`
	err := DB.QueryRow(statement, studentID).Scan(&avgGrade)

	if err != nil {
		log.Printf("Error getting average grade for student %s: %v", studentID, err)
		return 0, err
	}

	return avgGrade, nil
}
