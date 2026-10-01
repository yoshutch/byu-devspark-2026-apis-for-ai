package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const defaultRateLimitWindow = 30 * time.Second

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	clients map[string]rateLimitState
}

type rateLimitState struct {
	windowStarted time.Time
	requestCount  int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		return nil
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		clients: make(map[string]rateLimitState),
	}
}

func (limiter *rateLimiter) allow(clientID string, now time.Time) (bool, time.Duration) {
	limiter.mu.Lock()
	defer limiter.mu.Unlock()

	state, ok := limiter.clients[clientID]
	if !ok || now.Sub(state.windowStarted) >= limiter.window {
		limiter.clients[clientID] = rateLimitState{
			windowStarted: now,
			requestCount:  1,
		}
		return true, 0
	}

	if state.requestCount >= limiter.limit {
		return false, limiter.window - now.Sub(state.windowStarted)
	}

	state.requestCount++
	limiter.clients[clientID] = state
	return true, 0
}

func writeRateLimitProblem(r *http.Request, w http.ResponseWriter, retryAfter time.Duration) {
	retryAfterSeconds := int((retryAfter + time.Second - 1) / time.Second)
	if retryAfterSeconds < 1 {
		retryAfterSeconds = 1
	}
	w.Header().Set("Retry-After", strconv.Itoa(retryAfterSeconds))
	writeProblem(w, http.StatusTooManyRequests, problemDetails{
		Type:   problemTypeURL(r, "/problems/rate-limit-exceeded"),
		Title:  "Rate limit exceeded",
		Status: http.StatusTooManyRequests,
		Code:   "RATE_LIMIT_EXCEEDED",
		Detail: fmt.Sprintf("Too many requests from this client. Retry after %d seconds.", retryAfterSeconds),
	})
}

func parseRateLimit(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}

	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0, fmt.Errorf("DEMO_RATE_LIMIT must be a non-negative integer")
	}
	return limit, nil
}
