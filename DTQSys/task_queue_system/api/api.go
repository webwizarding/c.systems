package api

import (
    "context"
    "database/sql"
    "encoding/json"
    "net/http"
    "strings"
    "task_queue_system/auth"
    "task_queue_system/middleware"
    "task_queue_system/models"
    "task_queue_system/queue"
    "time"

    "github.com/go-chi/chi/v5"
    "github.com/go-playground/validator/v10"
    "github.com/google/uuid"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/rs/cors"
    log "github.com/sirupsen/logrus"
)

var (
    tasksReceived = prometheus.NewCounter(
        prometheus.CounterOpts{
            Name: "tasks_received_total",
            Help: "Total number of tasks received by the API",
        },
    )
    taskRequestDuration = prometheus.NewHistogram(
        prometheus.HistogramOpts{
            Name:    "task_request_duration_seconds",
            Help:    "Histogram of task request processing times",
            Buckets: prometheus.DefBuckets,
        },
    )
)

func init() {
    prometheus.MustRegister(tasksReceived)
    prometheus.MustRegister(taskRequestDuration)
}

type Server struct {
    Queue *queue.Queue
    DB    *sql.DB
}

var validate = validator.New()

func NewServer(queue *queue.Queue, db *sql.DB) *Server {
    return &Server{Queue: queue, DB: db}
}

func (s *Server) authMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
        claims, err := auth.ValidateToken(tokenStr)
        if err != nil {
            http.Error(w, err.Error(), http.StatusUnauthorized)
            return
        }

        // Add the username to the context
        ctx := context.WithValue(r.Context(), "username", claims.Username)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
    var creds models.Credentials
    err := json.NewDecoder(r.Body).Decode(&creds)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    // Validate credentials
    err = validate.Struct(creds)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    err = auth.RegisterUser(s.DB, creds.Username, creds.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
    var creds models.Credentials
    err := json.NewDecoder(r.Body).Decode(&creds)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    accessToken, refreshToken, err := auth.AuthenticateUser(s.DB, creds.Username, creds.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusUnauthorized)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "access_token":  accessToken,
        "refresh_token": refreshToken,
    })
}

func (s *Server) RefreshToken(w http.ResponseWriter, r *http.Request) {
    var request map[string]string
    err := json.NewDecoder(r.Body).Decode(&request)
    if err != nil {
        http.Error(w, "Invalid request payload", http.StatusBadRequest)
        return
    }

    refreshToken := request["refresh_token"]
    claims, err := auth.ValidateToken(refreshToken)
    if err != nil {
        http.Error(w, "Invalid refresh token", http.StatusUnauthorized)
        return
    }

    // Generate new tokens
    accessToken, newRefreshToken, err := auth.GenerateTokens(claims.Username)
    if err != nil {
        http.Error(w, "Failed to generate tokens", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(map[string]string{
        "access_token":  accessToken,
        "refresh_token": newRefreshToken,
    })
}

func (s *Server) CreateTask(w http.ResponseWriter, r *http.Request) {
    startTime := time.Now()

    var task models.Task
    if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    // Validate task
    err := validate.Struct(task)
    if err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    task.ID = uuid.New().String()
    task.Status = "pending"
    task.Created = time.Now()
    task.Retries = 0

    // Set default priority if not provided
    if task.Priority == 0 {
        task.Priority = 1
    }

    if err := s.Queue.Enqueue(task); err != nil {
        http.Error(w, "Failed to enqueue task", http.StatusInternalServerError)
        return
    }

    tasksReceived.Inc()
    duration := time.Since(startTime).Seconds()
    taskRequestDuration.Observe(duration)

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(task)
}

func (s *Server) GetActiveWorkers(w http.ResponseWriter, r *http.Request) {
    ctx := context.Background()
    workers, err := s.Queue.Client.HGetAll(ctx, "workers").Result()
    if err != nil {
        http.Error(w, "Failed to get active workers", http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(workers)
}

func (s *Server) GetTasks(w http.ResponseWriter, r *http.Request) {
    rows, err := s.DB.Query("SELECT task_id, data, status, created, retries, priority FROM tasks")
    if err != nil {
        http.Error(w, "Failed to get tasks", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var tasks []models.Task
    for rows.Next() {
        var task models.Task
        err := rows.Scan(&task.ID, &task.Data, &task.Status, &task.Created, &task.Retries, &task.Priority)
        if err != nil {
            http.Error(w, "Failed to parse tasks", http.StatusInternalServerError)
            return
        }
        tasks = append(tasks, task)
    }

    json.NewEncoder(w).Encode(tasks)
}

func (s *Server) Routes() http.Handler {
    r := chi.NewRouter()

    // Apply global middleware
    r.Use(middleware.RateLimiter)
    r.Use(middleware.SecurityHeaders)

    // CORS configuration
    c := cors.New(cors.Options{
        AllowedOrigins:   []string{"*"}, // Adjust as needed
        AllowedMethods:   []string{"GET", "POST"},
        AllowCredentials: true,
    })

    // Public endpoints
    r.Post("/register", s.RegisterUser)
    r.Post("/login", s.Login)
    r.Post("/refresh", s.RefreshToken)
    r.Handle("/metrics", prometheus.Handler())

    // Protected endpoints
    r.Group(func(r chi.Router) {
        r.Use(s.authMiddleware)
        r.Post("/tasks", s.CreateTask)
        r.Get("/tasks", s.GetTasks)
        r.Get("/workers", s.GetActiveWorkers)
    })

    handler := c.Handler(r)
    return handler
}
