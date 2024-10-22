package middleware

import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
)

var visitors = make(map[string]*rate.Limiter)
var mu sync.Mutex

func getVisitor(ip string) *rate.Limiter {
    mu.Lock()
    defer mu.Unlock()

    limiter, exists := visitors[ip]
    if !exists {
        limiter = rate.NewLimiter(1, 5) // 1 request per second, burst of 5
        visitors[ip] = limiter

        go func() {
            time.Sleep(3 * time.Minute)
            mu.Lock()
            delete(visitors, ip)
            mu.Unlock()
        }()
    }
    return limiter
}

func RateLimiter(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        ip := r.RemoteAddr
        limiter := getVisitor(ip)

        if !limiter.Allow() {
            http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
            return
        }
        next.ServeHTTP(w, r)
    })
}
