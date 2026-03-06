package ratelimit

import (
	"sync"
	"time"
)

// Represents a Token Bucket system
type TokenBucket struct {
	bucket     float64
	capacity   float64
	rate       float64   // per second
	lastAccess time.Time // stores last access timestamp
}

// Represents a Rate Limiter system
// Stores token buckets per-user
type RateLimiter struct {
	connections  map[string]*TokenBucket
	globalBucket *TokenBucket
	mu           sync.Mutex
}

// Returns a new RateLimiter struct
func NewRateLimiter() *RateLimiter {
	connections := make(map[string]*TokenBucket)
	bucket := TokenBucket{10, 10, 2, time.Now()} // fetch the rate limit variables from config file (TODO)
	rl := RateLimiter{connections: connections, globalBucket: &bucket, mu: sync.Mutex{}}

	// Launch TTL cleanup goroutine
	go rl.cleanUp(5)
	return &rl
}

// Periodically removes IPs who havent sent a request in `duration` amount of time
func (rl *RateLimiter) cleanUp(duration int) {
	period := time.Duration(duration) * time.Minute
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()

		for ip, tb := range rl.connections {
			if time.Since(tb.lastAccess) > period {
				delete(rl.connections, ip)
			}
		}

		rl.mu.Unlock()
	}
}

// Lazily refill bucket and check if rate limit has been exhausted
func (tb *TokenBucket) Allow() bool {
	now := time.Now()
	timeDifference := now.Sub(tb.lastAccess)
	tb.bucket += timeDifference.Seconds() * tb.rate

	if tb.bucket > tb.capacity {
		tb.bucket = tb.capacity
	}
	tb.lastAccess = now

	if tb.bucket >= 1 {
		tb.bucket--
		return true
	}

	return false
}

func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Check if global limit has been exhausted
	if !rl.globalBucket.Allow() {
		return false
	}

	//if user doesnt have a token bucket, create one and return true
	if rl.connections[ip] == nil {

		// remember to reduce bucket by 1 here!!!
		tb := TokenBucket{9, 10, 2, time.Now()} // fetch the rate limit variables from config file (TODO)
		rl.connections[ip] = &tb
		return true
	}

	// Check if rate limit has been exhausted or not (from per-ip bucket)
	return rl.connections[ip].Allow()
}
