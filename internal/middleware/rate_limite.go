package middleware

import (
	"net/http"
	"sync"
	"time"
	"golang.org/x/time/rate"
)

// Client holds the rate limiter for each visitor and the last time the visitor was seen
type Client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// IPRateLimiter manages rate limiting for different IP addresses
type IPRateLimiter struct {
	clients    map[string]*Client
	mu         sync.RWMutex
	rate       rate.Limit
	burst      int
	expiration time.Duration
}

// NewIPRateLimiter creates a new rate limiter instance
func NewIPRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		clients:    make(map[string]*Client),
		rate:       r,
		burst:      b,
		expiration: 1 * time.Hour, // Cleanup unused IPs after 1 hour
	}
}

// AddClient creates a new rate limiter for a client IP
func (rl *IPRateLimiter) AddClient(ip string) *rate.Limiter {
	limiter := rate.NewLimiter(rl.rate, rl.burst)
	rl.mu.Lock()
	rl.clients[ip] = &Client{
		limiter:  limiter,
		lastSeen: time.Now(),
	}
	rl.mu.Unlock()
	return limiter
}

// GetLimiter returns the rate limiter for a client IP
func (rl *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, exists := rl.clients[ip]
	if !exists {
		return rl.AddClient(ip)
	}

	// Update last seen time
	client.lastSeen = time.Now()
	return client.limiter
}

// CleanupStaleClients removes rate limiters for IPs that haven't been seen for a while
func (rl *IPRateLimiter) CleanupStaleClients() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	for ip, client := range rl.clients {
		if time.Since(client.lastSeen) > rl.expiration {
			delete(rl.clients, ip)
		}
	}
}

// Create a global rate limiter instance
var limiter = NewIPRateLimiter(1, 5) // 1 request per second with burst of 5

// RateLimitMiddleware is the middleware function to limit requests by IP
func RateLimitMiddleware(next http.Handler) http.Handler {
	go func() {
		for {
			time.Sleep(time.Hour)
			limiter.CleanupStaleClients()
		}
	}()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get IP address from request
		ip := r.RemoteAddr
		// For production, you might want to handle X-Forwarded-For or X-Real-IP headers
		if forwardedIP := r.Header.Get("X-Forwarded-For"); forwardedIP != "" {
			ip = forwardedIP
		}

		// Get rate limiter for this IP
		limiter := limiter.GetLimiter(ip)

		if !limiter.Allow() {
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
