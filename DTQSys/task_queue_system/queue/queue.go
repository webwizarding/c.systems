package queue

import (
    "context"
    "database/sql"
    "encoding/json"
    "task_queue_system/db"
    "task_queue_system/models"

    "github.com/go-redis/redis/v8"
)

var ctx = context.Background()

type Queue struct {
    Client *redis.Client
    db     *sql.DB
}

func NewQueue(redisAddr string, db *sql.DB) *Queue {
    client := redis.NewClient(&redis.Options{
        Addr: redisAddr,
    })
    return &Queue{Client: client, db: db}
}

func (q *Queue) Enqueue(task models.Task) error {
    // Save task to the database
    if err := db.InsertTask(q.db, task); err != nil {
        return err
    }

    // Enqueue the task in Redis
    data, err := json.Marshal(task)
    if err != nil {
        return err
    }

    var queueName string
    switch task.Priority {
    case 3:
        queueName = "high_task_queue"
    case 2:
        queueName = "medium_task_queue"
    default:
        queueName = "low_task_queue"
    }

    return q.Client.RPush(ctx, queueName, data).Err()
}

func (q *Queue) Dequeue() (*models.Task, error) {
    var task models.Task
    for _, queueName := range []string{"high_task_queue", "medium_task_queue", "low_task_queue"} {
        result, err := q.Client.LPop(ctx, queueName).Result()
        if err == redis.Nil {
            continue
        } else if err != nil {
            return nil, err
        }

        if err := json.Unmarshal([]byte(result), &task); err != nil {
            return nil, err
        }
        return &task, nil
    }
    return nil, redis.Nil
}
