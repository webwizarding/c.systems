package workers

import (
    "context"
    "database/sql"
    "math/rand"
    "task_queue_system/db"
    "task_queue_system/models"
    "task_queue_system/queue"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    log "github.com/sirupsen/logrus"
)

var (
    tasksProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "tasks_processed_total",
            Help: "Total number of tasks processed",
        },
        []string{"status"},
    )
    taskProcessingTime = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "task_processing_duration_seconds",
            Help:    "Histogram of task processing times",
            Buckets: prometheus.DefBuckets,
        },
    )
)

func init() {
    // Register metrics
    prometheus.MustRegister(tasksProcessed)
    prometheus.MustRegister(taskProcessingTime)
}

const MaxRetries = 3

type Worker struct {
    ID    string
    Queue *queue.Queue
    db    *sql.DB
}

func NewWorker(id string, queue *queue.Queue, db *sql.DB) *Worker {
    return &Worker{ID: id, Queue: queue, db: db}
}

func (w *Worker) Register() {
    // Register the worker in Redis
    ctx := context.Background()
    w.Queue.Client.HSet(ctx, "workers", w.ID, "active")
    log.WithField("worker", w.ID).Info("Worker registered")
}

func (w *Worker) Deregister() {
    // Deregister the worker from Redis
    ctx := context.Background()
    w.Queue.Client.HDel(ctx, "workers", w.ID)
    log.WithField("worker", w.ID).Info("Worker deregistered")
}

func (w *Worker) Start(stopChan chan struct{}) {
    w.Register()
    defer w.Deregister()

    for {
        select {
        case <-stopChan:
            log.WithField("worker", w.ID).Info("Worker stopping gracefully")
            return
        default:
            task, err := w.Queue.Dequeue()
            if err != nil {
                time.Sleep(1 * time.Second)
                continue
            }

            w.processTask(task)
        }
    }
}

func (w *Worker) processTask(task *models.Task) {
    startTime := time.Now()
    log.WithFields(log.Fields{
        "worker": w.ID,
        "task":   task.ID,
    }).Info("Processing task")

    // Simulate task processing with a random failure
    if rand.Intn(4) == 0 { // 25% chance to fail
        log.WithFields(log.Fields{
            "worker": w.ID,
            "task":   task.ID,
        }).Warn("Failed to process task")
        task.Retries++

        if task.Retries < MaxRetries {
            if err := w.Queue.Enqueue(*task); err != nil {
                log.WithFields(log.Fields{
                    "worker": w.ID,
                    "task":   task.ID,
                }).Error("Failed to re-enqueue task")
            }
        } else {
            task.Status = "failed"
            db.UpdateTaskStatus(w.db, task.ID, "failed")
            tasksProcessed.WithLabelValues("failed").Inc()
        }
    } else {
        // Simulate task processing time
        time.Sleep(2 * time.Second)
        task.Status = "completed"
        db.UpdateTaskStatus(w.db, task.ID, "completed")
        tasksProcessed.WithLabelValues("completed").Inc()
        log.WithFields(log.Fields{
            "worker": w.ID,
            "task":   task.ID,
        }).Info("Completed task")
    }

    duration := time.Since(startTime).Seconds()
    taskProcessingTime.Observe(duration)
}
