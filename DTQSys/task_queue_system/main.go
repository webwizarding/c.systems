package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "task_queue_system/api"
    "task_queue_system/db"
    "task_queue_system/queue"
    "task_queue_system/workers"

    "github.com/joho/godotenv"
    logrus "github.com/sirupsen/logrus"
)

func init() {
    // Configure logrus
    logrus.SetFormatter(&logrus.JSONFormatter{})
    logrus.SetOutput(os.Stdout)
    logrus.SetLevel(logrus.InfoLevel)
}

func main() {
    // Load environment variables
    if err := godotenv.Load(); err != nil {
        logrus.Info("No .env file found, using system environment variables")
    }

    // Initialize the database
    database, err := db.OpenDB()
    if err != nil {
        logrus.Fatalf("Failed to connect to database: %v", err)
    }
    defer database.Close()

    // Initialize the queue
    redisAddr := os.Getenv("REDIS_ADDR")
    taskQueue := queue.NewQueue(redisAddr, database)

    // Number of workers to start
    numWorkers := 5

    // WaitGroup to wait for all workers to finish
    var wg sync.WaitGroup

    // Channel to signal workers to stop
    stopChan := make(chan struct{})

    // Start multiple workers
    for i := 0; i < numWorkers; i++ {
        wg.Add(1)
        workerID := fmt.Sprintf("worker-%d", i+1)
        worker := workers.NewWorker(workerID, taskQueue, database)
        go func() {
            defer wg.Done()
            worker.Start(stopChan)
        }()
    }

    // Set up the API server
    server := api.NewServer(taskQueue, database)

    // Expose the metrics endpoint
    http.Handle("/metrics", server.Routes())

    // Start the HTTPS server
    go func() {
        logrus.Info("Server started at :8443")
        if err := http.ListenAndServeTLS(":8443", "cert/cert.pem", "cert/key.pem", server.Routes()); err != nil {
            logrus.Fatalf("HTTPS server error: %v", err)
        }
    }()

    // Wait for an interrupt signal to gracefully shut down
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
    <-sigs
    logrus.Info("Shutdown signal received")

    // Signal workers to stop
    close(stopChan)

    // Wait for all workers to finish
    wg.Wait()
    logrus.Info("All workers have stopped")
}
