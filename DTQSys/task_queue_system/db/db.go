package db

import (
    "database/sql"
    "fmt"
    "os"
    "task_queue_system/models"

    _ "github.com/lib/pq"
)

func OpenDB() (*sql.DB, error) {
    host := os.Getenv("DB_HOST")
    port := os.Getenv("DB_PORT")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    dbname := os.Getenv("DB_NAME")

    psqlInfo := fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        host, port, user, password, dbname)
    db, err := sql.Open("postgres", psqlInfo)
    if err != nil {
        return nil, err
    }
    return db, db.Ping()
}

func InsertTask(db *sql.DB, task models.Task) error {
    sqlStatement := `
        INSERT INTO tasks (task_id, data, status, created, retries, priority)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (task_id) DO NOTHING`
    _, err := db.Exec(sqlStatement,
        task.ID, task.Data, task.Status, task.Created,
        task.Retries, task.Priority)
    return err
}

func UpdateTaskStatus(db *sql.DB, taskID, status string) error {
    sqlStatement := `
        UPDATE tasks SET status = $1 WHERE task_id = $2`
    _, err := db.Exec(sqlStatement, status, taskID)
    return err
}
