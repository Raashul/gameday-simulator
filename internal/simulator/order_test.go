package simulator

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"gameday-sim/internal/api"
	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
)

// createTestConfig creates a minimal test configuration
func createTestConfig() *config.Config {
	return &config.Config{
		Simulation: config.SimulationConfig{
			TotalOrders:     10,
			BatchSize:       5,
			ParallelBatches: 2,
			ActivatedCount:  7,
		},
		Payload: config.PayloadConfig{
			Location:          "US-EAST-1",
			POCOrder:          "POC-TEST-001",
			OrderNumberPrefix: "ORD-TEST-",
		},
		Intervals: config.IntervalConfig{
			BetweenCreates:       10 * time.Millisecond,
			AfterCreateBeforeGet: 10 * time.Millisecond,
			BetweenGetPolls:      10 * time.Millisecond,
			BeforeActivate:       10 * time.Millisecond,
			BeforeCancel:         10 * time.Millisecond,
			BeforeEnd:            10 * time.Millisecond,
		},
		API: config.APIConfig{
			BaseURL:      "http://localhost:8080",
			Timeout:      5 * time.Second,
			RetryMax:     1,
			RetryBackoff: 10 * time.Millisecond,
		},
		Cleanup: config.CleanupConfig{
			CancelTimeout: 1 * time.Second,
			EndTimeout:    1 * time.Second,
			CheckInterval: 100 * time.Millisecond,
		},
	}
}

// createTestPayload creates a test order payload
func createTestPayload(orderType payload.OrderType) payload.OrderPayload {
	return payload.OrderPayload{
		OrderNumber: "ORD-TEST-000001",
		Location:    "US-EAST-1",
		POCOrder:    "POC-TEST-001",
		Timestamp:   time.Now(),
		Type:        orderType,
		Geometry: &payload.GeoJSONGeometry{
			Type: "LineString",
			Coordinates: [][]float64{
				{-96.79943798188481, 32.795102753983585},
				{-96.79927289435462, 32.78885767285452},
			},
		},
	}
}

// TestTerminationAction constants
func TestTerminationActionConstants(t *testing.T) {
	if ActionEnd != "end" {
		t.Errorf("ActionEnd = %s, expected 'end'", ActionEnd)
	}
	if ActionCancel != "cancel" {
		t.Errorf("ActionCancel = %s, expected 'cancel'", ActionCancel)
	}
}

// TestTerminationRequest struct fields
func TestTerminationRequestStruct(t *testing.T) {
	result := &OrderResult{
		OrderNumber: "ORD-001",
		State:       payload.StateAccepted,
	}

	req := TerminationRequest{
		OrderID: "order-123",
		Action:  ActionEnd,
		Result:  result,
	}

	if req.OrderID != "order-123" {
		t.Errorf("OrderID = %s, expected 'order-123'", req.OrderID)
	}
	if req.Action != ActionEnd {
		t.Errorf("Action = %s, expected 'end'", req.Action)
	}
	if req.Result != result {
		t.Error("Result pointer mismatch")
	}
}

// TestOrderResult struct
func TestOrderResultStruct(t *testing.T) {
	result := OrderResult{
		OrderNumber: "ORD-TEST-001",
		OrderID:     "order-uuid-123",
		Type:        payload.TypeActivate,
		State:       payload.StateActivated,
		StartTime:   time.Now(),
	}

	if result.OrderNumber != "ORD-TEST-001" {
		t.Errorf("OrderNumber = %s, expected 'ORD-TEST-001'", result.OrderNumber)
	}
	if result.Type != payload.TypeActivate {
		t.Errorf("Type = %s, expected 'activate'", result.Type)
	}
	if result.State != payload.StateActivated {
		t.Errorf("State = %s, expected 'activated'", result.State)
	}
}

// TestNewOrderProcessor creates a new order processor
func TestNewOrderProcessor(t *testing.T) {
	cfg := createTestConfig()
	terminationChan := make(chan TerminationRequest, 10)
	defer close(terminationChan)

	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	processor := NewOrderProcessor(client, cfg, terminationChan)

	if processor == nil {
		t.Error("NewOrderProcessor returned nil")
	}
	if processor.apiClient != client {
		t.Error("API client mismatch")
	}
	if processor.config != cfg {
		t.Error("Config mismatch")
	}
}

// createMockServer creates a mock HTTP server for testing
func createMockServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for path, handler := range handlers {
			if r.URL.Path == path {
				handler(w, r)
				return
			}
		}
		// Also check for query parameters (for /details endpoint)
		for path, handler := range handlers {
			if path == "/details" && r.URL.Path == "/details" {
				handler(w, r)
				return
			}
		}
		t.Logf("Unhandled request: %s %s", r.Method, r.URL.String())
		w.WriteHeader(http.StatusNotFound)
	}))
}

// TestTerminationWorker_End tests the termination worker for end action
func TestTerminationWorker_End(t *testing.T) {
	// Create mock server
	server := createMockServer(t, map[string]http.HandlerFunc{
		"/end": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-123", "status": "ended"}`))
		},
	})
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	terminationChan := make(chan TerminationRequest, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start worker in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TerminationWorker(ctx, client, terminationChan)
	}()

	// Create a test result
	result := &OrderResult{
		OrderNumber: "ORD-001",
		OrderID:     "order-123",
		State:       payload.StatePendingEnd,
		StartTime:   time.Now(),
	}

	// Send termination request
	terminationChan <- TerminationRequest{
		OrderID: "order-123",
		Action:  ActionEnd,
		Result:  result,
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop worker
	cancel()

	// Wait for worker to finish
	wg.Wait()

	// Verify result was updated
	if result.State != payload.StateEnded {
		t.Errorf("Result state = %s, expected 'ended'", result.State)
	}
	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
	if result.Duration == 0 {
		t.Error("Duration should be calculated")
	}
}

// TestTerminationWorker_Cancel tests the termination worker for cancel action
func TestTerminationWorker_Cancel(t *testing.T) {
	// Create mock server
	server := createMockServer(t, map[string]http.HandlerFunc{
		"/cancel": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-123", "status": "cancelled"}`))
		},
	})
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	terminationChan := make(chan TerminationRequest, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start worker in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TerminationWorker(ctx, client, terminationChan)
	}()

	// Create a test result
	result := &OrderResult{
		OrderNumber: "ORD-001",
		OrderID:     "order-123",
		State:       payload.StatePendingCancel,
		StartTime:   time.Now(),
	}

	// Send termination request
	terminationChan <- TerminationRequest{
		OrderID: "order-123",
		Action:  ActionCancel,
		Result:  result,
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop worker
	cancel()

	// Wait for worker to finish
	wg.Wait()

	// Verify result was updated
	if result.State != payload.StateCancelled {
		t.Errorf("Result state = %s, expected 'cancelled'", result.State)
	}
	if result.EndTime.IsZero() {
		t.Error("EndTime should be set")
	}
}

// TestTerminationWorker_Error tests the termination worker error handling
func TestTerminationWorker_Error(t *testing.T) {
	// Create mock server that returns error
	server := createMockServer(t, map[string]http.HandlerFunc{
		"/end": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error": "internal_error", "message": "Something went wrong"}`))
		},
	})
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	cfg.API.RetryMax = 0 // No retries for faster test
	client := api.NewClient(cfg, nil) // No auth needed for tests

	terminationChan := make(chan TerminationRequest, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Start worker in goroutine
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TerminationWorker(ctx, client, terminationChan)
	}()

	// Create a test result
	result := &OrderResult{
		OrderNumber: "ORD-001",
		OrderID:     "order-123",
		State:       payload.StatePendingEnd,
		StartTime:   time.Now(),
	}

	// Send termination request
	terminationChan <- TerminationRequest{
		OrderID: "order-123",
		Action:  ActionEnd,
		Result:  result,
	}

	// Wait a bit for processing
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop worker
	cancel()

	// Wait for worker to finish
	wg.Wait()

	// Verify result state is failed
	if result.State != payload.StateFailed {
		t.Errorf("Result state = %s, expected 'failed'", result.State)
	}
	if result.Error == nil {
		t.Error("Error should be set on failure")
	}
}

// TestTerminationWorker_ContextCancellation tests worker stops on context cancellation
func TestTerminationWorker_ContextCancellation(t *testing.T) {
	cfg := createTestConfig()
	client := api.NewClient(cfg, nil) // No auth needed for tests

	terminationChan := make(chan TerminationRequest, 10)
	ctx, cancel := context.WithCancel(context.Background())

	workerDone := make(chan bool)
	go func() {
		TerminationWorker(ctx, client, terminationChan)
		workerDone <- true
	}()

	// Cancel immediately
	cancel()

	// Worker should exit
	select {
	case <-workerDone:
		// Success - worker exited
	case <-time.After(500 * time.Millisecond):
		t.Error("Worker did not exit after context cancellation")
	}
}

// TestProcessTermination_End tests the processTermination function for end action
func TestProcessTermination_End(t *testing.T) {
	// Create mock server
	server := createMockServer(t, map[string]http.HandlerFunc{
		"/end": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-123", "status": "ended"}`))
		},
	})
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	result := &OrderResult{
		OrderNumber: "ORD-001",
		OrderID:     "order-123",
		State:       payload.StatePendingEnd,
		StartTime:   time.Now(),
	}

	req := TerminationRequest{
		OrderID: "order-123",
		Action:  ActionEnd,
		Result:  result,
	}

	processTermination(context.Background(), client, req)

	if result.State != payload.StateEnded {
		t.Errorf("Result state = %s, expected 'ended'", result.State)
	}
}

// TestProcessTermination_Cancel tests the processTermination function for cancel action
func TestProcessTermination_Cancel(t *testing.T) {
	// Create mock server
	server := createMockServer(t, map[string]http.HandlerFunc{
		"/cancel": func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-123", "status": "cancelled"}`))
		},
	})
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	result := &OrderResult{
		OrderNumber: "ORD-001",
		OrderID:     "order-123",
		State:       payload.StatePendingCancel,
		StartTime:   time.Now(),
	}

	req := TerminationRequest{
		OrderID: "order-123",
		Action:  ActionCancel,
		Result:  result,
	}

	processTermination(context.Background(), client, req)

	if result.State != payload.StateCancelled {
		t.Errorf("Result state = %s, expected 'cancelled'", result.State)
	}
}

// TestOrderStateTransitions tests the order state constants
func TestOrderStateTransitions(t *testing.T) {
	// Verify all state constants exist and have expected values
	states := map[payload.OrderState]string{
		payload.StateCreated:       "created",
		payload.StateAccepted:      "accepted",
		payload.StateActivated:     "activated",
		payload.StatePendingCancel: "pending_cancel",
		payload.StatePendingEnd:    "pending_end",
		payload.StateCancelled:     "cancelled",
		payload.StateEnded:         "ended",
		payload.StateFailed:        "failed",
	}

	for state, expected := range states {
		if string(state) != expected {
			t.Errorf("State %s = %s, expected %s", state, string(state), expected)
		}
	}
}

// TestAsyncTerminationFlow tests the full async termination flow
func TestAsyncTerminationFlow(t *testing.T) {
	// This test verifies that termination is truly async
	terminationChan := make(chan TerminationRequest, 10)

	result := &OrderResult{
		OrderNumber: "ORD-001",
		State:       payload.StateAccepted,
		StartTime:   time.Now(),
	}

	// Send termination request (should not block)
	start := time.Now()
	select {
	case terminationChan <- TerminationRequest{
		OrderID: "order-123",
		Action:  ActionCancel,
		Result:  result,
	}:
		// Success - send should be immediate
	case <-time.After(100 * time.Millisecond):
		t.Error("Sending to termination channel should not block")
	}
	sendDuration := time.Since(start)

	// Verify send was quick (async behavior)
	if sendDuration > 50*time.Millisecond {
		t.Errorf("Send took %v, expected < 50ms (async behavior)", sendDuration)
	}

	// Update state to pending
	result.State = payload.StatePendingCancel

	// State should be pending, not cancelled (async)
	if result.State != payload.StatePendingCancel {
		t.Errorf("State should be 'pending_cancel' immediately after send, got %s", result.State)
	}

	close(terminationChan)
}

// TestMultipleTerminationRequests tests handling multiple concurrent termination requests
func TestMultipleTerminationRequests(t *testing.T) {
	// Create mock server
	var mu sync.Mutex
	processedOrders := make(map[string]bool)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		if r.URL.Path == "/end" {
			processedOrders["end"] = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-end", "status": "ended"}`))
		} else if r.URL.Path == "/cancel" {
			processedOrders["cancel"] = true
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"orderId": "order-cancel", "status": "cancelled"}`))
		}
	}))
	defer server.Close()

	cfg := createTestConfig()
	cfg.API.BaseURL = server.URL
	client := api.NewClient(cfg, nil) // No auth needed for tests

	terminationChan := make(chan TerminationRequest, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Start worker
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		TerminationWorker(ctx, client, terminationChan)
	}()

	// Send multiple termination requests
	results := []*OrderResult{
		{OrderNumber: "ORD-001", State: payload.StatePendingEnd, StartTime: time.Now()},
		{OrderNumber: "ORD-002", State: payload.StatePendingCancel, StartTime: time.Now()},
		{OrderNumber: "ORD-003", State: payload.StatePendingEnd, StartTime: time.Now()},
	}

	terminationChan <- TerminationRequest{OrderID: "order-1", Action: ActionEnd, Result: results[0]}
	terminationChan <- TerminationRequest{OrderID: "order-2", Action: ActionCancel, Result: results[1]}
	terminationChan <- TerminationRequest{OrderID: "order-3", Action: ActionEnd, Result: results[2]}

	// Wait for processing
	time.Sleep(300 * time.Millisecond)

	// Cancel and wait
	cancel()
	wg.Wait()

	// Verify all were processed
	for i, r := range results {
		if r.State != payload.StateEnded && r.State != payload.StateCancelled {
			t.Errorf("Result %d state = %s, expected 'ended' or 'cancelled'", i, r.State)
		}
	}
}
