package utils

import (
	"sync"
	"time"
)

// Metrics tracks simulation metrics
type Metrics struct {
	mu sync.RWMutex

	// API call metrics
	apiCalls       map[string]int
	apiSuccesses   map[string]int
	apiFailures    map[string]int
	apiDurations   map[string][]time.Duration

	// Order state metrics
	orderStates    map[string]int

	// Batch metrics
	batchesStarted   int
	batchesCompleted int
	batchesFailed    int
}

// NewMetrics creates a new metrics tracker
func NewMetrics() *Metrics {
	return &Metrics{
		apiCalls:     make(map[string]int),
		apiSuccesses: make(map[string]int),
		apiFailures:  make(map[string]int),
		apiDurations: make(map[string][]time.Duration),
		orderStates:  make(map[string]int),
	}
}

// RecordAPICall records an API call
func (m *Metrics) RecordAPICall(endpoint string, success bool, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.apiCalls[endpoint]++
	if success {
		m.apiSuccesses[endpoint]++
	} else {
		m.apiFailures[endpoint]++
	}

	m.apiDurations[endpoint] = append(m.apiDurations[endpoint], duration)
}

// RecordOrderState records an order state transition
func (m *Metrics) RecordOrderState(state string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.orderStates[state]++
}

// RecordBatchStarted increments batch started counter
func (m *Metrics) RecordBatchStarted() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.batchesStarted++
}

// RecordBatchCompleted increments batch completed counter
func (m *Metrics) RecordBatchCompleted(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if success {
		m.batchesCompleted++
	} else {
		m.batchesFailed++
	}
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := MetricsSnapshot{
		APICalls:         make(map[string]APIMetrics),
		OrderStates:      make(map[string]int),
		BatchesStarted:   m.batchesStarted,
		BatchesCompleted: m.batchesCompleted,
		BatchesFailed:    m.batchesFailed,
	}

	// Copy API metrics
	for endpoint := range m.apiCalls {
		snapshot.APICalls[endpoint] = APIMetrics{
			TotalCalls:      m.apiCalls[endpoint],
			SuccessfulCalls: m.apiSuccesses[endpoint],
			FailedCalls:     m.apiFailures[endpoint],
			AvgDuration:     calculateAverage(m.apiDurations[endpoint]),
			MinDuration:     calculateMin(m.apiDurations[endpoint]),
			MaxDuration:     calculateMax(m.apiDurations[endpoint]),
		}
	}

	// Copy order state metrics
	for state, count := range m.orderStates {
		snapshot.OrderStates[state] = count
	}

	return snapshot
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	APICalls         map[string]APIMetrics
	OrderStates      map[string]int
	BatchesStarted   int
	BatchesCompleted int
	BatchesFailed    int
}

// APIMetrics represents metrics for a specific API endpoint
type APIMetrics struct {
	TotalCalls      int
	SuccessfulCalls int
	FailedCalls     int
	AvgDuration     time.Duration
	MinDuration     time.Duration
	MaxDuration     time.Duration
}

// Helper functions for calculating duration statistics
func calculateAverage(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	var total time.Duration
	for _, d := range durations {
		total += d
	}

	return total / time.Duration(len(durations))
}

func calculateMin(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	min := durations[0]
	for _, d := range durations[1:] {
		if d < min {
			min = d
		}
	}

	return min
}

func calculateMax(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	max := durations[0]
	for _, d := range durations[1:] {
		if d > max {
			max = d
		}
	}

	return max
}
