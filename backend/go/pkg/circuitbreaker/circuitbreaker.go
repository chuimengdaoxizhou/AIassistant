package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// State represents the state of the circuit breaker.
type State int

const (
	// Closed is the initial state where requests are allowed.
	Closed State = iota
	// Open state is when the circuit has tripped and requests are blocked.
	Open
	// HalfOpen is a state where a limited number of trial requests are allowed to test the system's recovery.
	HalfOpen
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case Closed:
		return "Closed"
	case Open:
		return "Open"
	case HalfOpen:
		return "Half-Open"
	default:
		return "Unknown"
	}
}

var (
	// ErrCircuitOpen is returned when the circuit breaker is in the Open state.
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreaker is the interface for the circuit breaker pattern.
type CircuitBreaker interface {
	// Execute runs the given request if the circuit breaker is closed or half-open.
	Execute(req func() (interface{}, error)) (interface{}, error)
	// State returns the current state of the circuit breaker.
	State() State
}

// options holds the configuration for a circuitBreaker.
type options struct {
	failureThreshold     uint32        // Number of failures to trip the circuit.
	successThreshold     uint32        // Number of successes in HalfOpen state to close the circuit.
	timeout              time.Duration // Duration to wait in Open state before transitioning to HalfOpen.
	consecutiveSuccesses uint32        // Current count of consecutive successes.
	consecutiveFailures  uint32        // Current count of consecutive failures.
	lastErrorTime        time.Time     // Time when the circuit was opened.
	state                State
	mutex                sync.Mutex
}

// New creates a new circuitBreaker with the specified settings.
// failureThreshold: The number of consecutive failures required to open the circuit.
// successThreshold: The number of consecutive successes in the half-open state required to close the circuit.
// timeout: The duration the circuit remains open before transitioning to half-open.
func New(failureThreshold, successThreshold uint32, timeout time.Duration) CircuitBreaker {
	return &options{
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		timeout:          timeout,
		state:            Closed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *options) State() State {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.state
}

// Execute wraps the execution of a function with the circuit breaker logic.
func (cb *options) Execute(req func() (interface{}, error)) (interface{}, error) {
	cb.mutex.Lock()

	// Check if we should transition from Open to HalfOpen
	if cb.state == Open && time.Since(cb.lastErrorTime) > cb.timeout {
		cb.state = HalfOpen
		cb.consecutiveSuccesses = 0
	}

	// Handle request based on state
	switch cb.state {
	case Open:
		cb.mutex.Unlock()
		return nil, ErrCircuitOpen
	case HalfOpen:
		cb.mutex.Unlock()
		res, err := req()
		if err != nil {
			cb.onFailure()
			return nil, err
		}
		cb.onSuccess()
		return res, nil
	case Closed:
		cb.mutex.Unlock()
		res, err := req()
		if err != nil {
			cb.onFailure()
			return nil, err
		}
		cb.onSuccess()
		return res, nil
	default:
		cb.mutex.Unlock()
		return nil, errors.New("unknown circuit breaker state")
	}
}

// onSuccess handles the logic when a request succeeds.
func (cb *options) onSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case HalfOpen:
		cb.consecutiveSuccesses++
		if cb.consecutiveSuccesses >= cb.successThreshold {
			cb.reset()
		}
	case Closed:
		cb.resetFailures()
	}
}

// onFailure handles the logic when a request fails.
func (cb *options) onFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	switch cb.state {
	case HalfOpen:
		cb.trip()
	case Closed:
		cb.consecutiveFailures++
		if cb.consecutiveFailures >= cb.failureThreshold {
			cb.trip()
		}
	}
}

// trip opens the circuit.
func (cb *options) trip() {
	cb.state = Open
	cb.lastErrorTime = time.Now()
	cb.resetFailures()
	cb.consecutiveSuccesses = 0
}

// reset closes the circuit and resets all counters.
func (cb *options) reset() {
	cb.state = Closed
	cb.resetFailures()
	cb.consecutiveSuccesses = 0
}

// resetFailures resets the failure counter.
func (cb *options) resetFailures() {
	cb.consecutiveFailures = 0
}
