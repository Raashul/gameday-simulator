package simulator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gameday-sim/internal/api"
	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
)

// BatchProcessor handles parallel batch processing
type BatchProcessor struct {
	apiClient       *api.Client
	config          *config.Config
	orderProcessor  *OrderProcessor
	terminationChan chan TerminationRequest
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(apiClient *api.Client, cfg *config.Config) *BatchProcessor {
	terminationChan := make(chan TerminationRequest, 1000)

	return &BatchProcessor{
		apiClient:       apiClient,
		config:          cfg,
		terminationChan: terminationChan,
		orderProcessor:  NewOrderProcessor(apiClient, cfg, terminationChan),
	}
}

// StartTerminationWorker starts the background worker for processing terminations
func (bp *BatchProcessor) StartTerminationWorker(ctx context.Context) {
	go TerminationWorker(ctx, bp.apiClient, bp.terminationChan)
}

// Close closes the termination channel
func (bp *BatchProcessor) Close() {
	close(bp.terminationChan)
}

// ProcessBatches processes all batches in parallel
func (bp *BatchProcessor) ProcessBatches(ctx context.Context, batches []payload.Batch) (*SimulationResult, error) {
	result := &SimulationResult{
		StartTime:    time.Now(),
		BatchResults: make([]BatchResult, 0, len(batches)),
	}

	// Channel to collect batch results
	resultsChan := make(chan BatchResult, len(batches))
	errorsChan := make(chan error, len(batches))

	// WaitGroup to track batch completion
	var wg sync.WaitGroup

	// Semaphore to limit parallel batches
	semaphore := make(chan struct{}, bp.config.Simulation.ParallelBatches)

	// Launch batch processors
	for _, batch := range batches {
		wg.Add(1)
		go func(b payload.Batch) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case <-ctx.Done():
				errorsChan <- ctx.Err()
				return
			case semaphore <- struct{}{}:
			}

			// Process batch
			batchResult := bp.processSingleBatch(ctx, b)
			resultsChan <- batchResult

			// Release semaphore
			<-semaphore
		}(batch)
	}

	// Close channels when all batches complete
	go func() {
		wg.Wait()
		close(resultsChan)
		close(errorsChan)
	}()

	// Collect results
	for batchResult := range resultsChan {
		result.BatchResults = append(result.BatchResults, batchResult)
		result.TotalOrders += batchResult.TotalOrders
		result.SuccessfulOrders += batchResult.SuccessfulOrders
		result.FailedOrders += batchResult.FailedOrders
	}

	// Check for errors
	for err := range errorsChan {
		if err != nil {
			return result, fmt.Errorf("batch processing error: %w", err)
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// processSingleBatch processes a single batch sequentially
func (bp *BatchProcessor) processSingleBatch(ctx context.Context, batch payload.Batch) BatchResult {
	result := BatchResult{
		BatchID:      batch.ID,
		StartTime:    time.Now(),
		TotalOrders:  len(batch.Payloads),
		OrderResults: make([]OrderResult, 0, len(batch.Payloads)),
	}

	// Process each payload in the batch sequentially
	for i, pl := range batch.Payloads {
		// Check context cancellation
		if ctx.Err() != nil {
			result.FailedOrders++
			continue
		}

		// Process the order
		orderResult, err := bp.orderProcessor.ProcessOrder(ctx, pl)
		if err != nil {
			result.FailedOrders++
		} else {
			result.SuccessfulOrders++
		}

		result.OrderResults = append(result.OrderResults, *orderResult)

		// Wait between creates (except for last item)
		if i < len(batch.Payloads)-1 {
			select {
			case <-ctx.Done():
				return result
			case <-time.After(bp.config.Intervals.BetweenCreates):
			}
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result
}

// BatchResult represents the result of processing a single batch
type BatchResult struct {
	BatchID          int
	TotalOrders      int
	SuccessfulOrders int
	FailedOrders     int
	OrderResults     []OrderResult
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
}

// SimulationResult represents the overall simulation result
type SimulationResult struct {
	TotalOrders      int
	SuccessfulOrders int
	FailedOrders     int
	BatchResults     []BatchResult
	StartTime        time.Time
	EndTime          time.Time
	Duration         time.Duration
}

// GetStats returns statistics about the simulation
func (sr *SimulationResult) GetStats() map[string]interface{} {
	activatedCount := 0
	cancelledCount := 0
	endedCount := 0
	failedCount := 0
	pendingCancelCount := 0
	pendingEndCount := 0

	var totalDuration time.Duration
	var avgDuration time.Duration

	orderCount := 0
	for _, batchResult := range sr.BatchResults {
		for _, orderResult := range batchResult.OrderResults {
			orderCount++
			totalDuration += orderResult.Duration

			switch orderResult.State {
			case payload.StateActivated:
				activatedCount++
			case payload.StateEnded:
				endedCount++
			case payload.StateCancelled:
				cancelledCount++
			case payload.StatePendingCancel:
				pendingCancelCount++
			case payload.StatePendingEnd:
				pendingEndCount++
			case payload.StateFailed:
				failedCount++
			}
		}
	}

	if orderCount > 0 {
		avgDuration = totalDuration / time.Duration(orderCount)
	}

	return map[string]interface{}{
		"totalOrders":       sr.TotalOrders,
		"successfulOrders":  sr.SuccessfulOrders,
		"failedOrders":      sr.FailedOrders,
		"activatedOrders":   activatedCount,
		"endedOrders":       endedCount,
		"cancelledOrders":   cancelledCount,
		"pendingCancelOrds": pendingCancelCount,
		"pendingEndOrds":    pendingEndCount,
		"totalDuration":     sr.Duration.String(),
		"avgOrderDuration":  avgDuration.String(),
		"totalBatches":      len(sr.BatchResults),
	}
}
