package main

import (
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

type visitor struct {
	count     int
	lastReset time.Time
}

var GlobalLimiter = &RateLimiter{
	visitors: make(map[string]*visitor),
}

func (rl *RateLimiter) Allow(ip string, limit int) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	v, exists := rl.visitors[ip]
	now := time.Now()

	if !exists {
		rl.visitors[ip] = &visitor{
			count:     1,
			lastReset: now,
		}
		return true
	}

	if now.Sub(v.lastReset) > time.Second {
		v.count = 1
		v.lastReset = now
		return true
	}

	if v.count >= limit {
		return false
	}

	v.count++
	return true
}

func (rl *RateLimiter) CleanupLoop() {
	for {
		time.Sleep(10 * time.Minute)
		rl.mu.Lock()
		for ip, v := range rl.visitors {
			if time.Since(v.lastReset) > 10*time.Minute {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}
