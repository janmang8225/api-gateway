package breaker

import (
	"fmt"
	"sync"
	"time"
)

type State int

const (
	StateClosed   State = iota // healthy, requests flow normally
	StateOpen                  // unhealthy, requests blocked
	StateHalfOpen              // testing, one request allowed through
)

type CircuitBreaker struct {
	mu              sync.Mutex
	state           State
	failures        int
	failureThreshold int
	successThreshold int
	successes       int
	cooldown        time.Duration
	openedAt        time.Time
}

func New(failureThreshold int, successThreshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		cooldown:         cooldown,
	}
}

func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateClosed {
		return true
	}

	if cb.state == StateOpen {
		if time.Since(cb.openedAt) >= cb.cooldown {
			fmt.Println("circuit breaker: moving to half-open")
			cb.state = StateHalfOpen
			cb.successes = 0
			return true
		}
		return false
	}

	// StateHalfOpen — allow one request through at a time
	if cb.state == StateHalfOpen {
		return true
	}

	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.successes++
		if cb.successes >= cb.successThreshold {
			fmt.Println("circuit breaker: closing — backend recovered")
			cb.state = StateClosed
			cb.failures = 0
		}
		return
	}

	// reset failure count on success in closed state
	if cb.state == StateClosed {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++

	if cb.failures >= cb.failureThreshold {
		fmt.Println("circuit breaker: opening — too many failures")
		cb.state = StateOpen
		cb.openedAt = time.Now()
	}
}

func (cb *CircuitBreaker) State() State {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}