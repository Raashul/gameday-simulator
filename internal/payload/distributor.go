package payload

import (
	"fmt"
)

// Batch represents a collection of payloads to be processed together
type Batch struct {
	ID       int
	Payloads []OrderPayload
}

// Distributor handles distribution of payloads into batches
type Distributor struct {
	batchSize int
}

// NewDistributor creates a new payload distributor
func NewDistributor(batchSize int) *Distributor {
	return &Distributor{
		batchSize: batchSize,
	}
}

// Distribute divides payloads into batches
func (d *Distributor) Distribute(payloads []OrderPayload) []Batch {
	if len(payloads) == 0 {
		return nil
	}

	// Calculate number of batches needed
	numBatches := (len(payloads) + d.batchSize - 1) / d.batchSize
	batches := make([]Batch, 0, numBatches)

	for i := 0; i < len(payloads); i += d.batchSize {
		end := i + d.batchSize
		if end > len(payloads) {
			end = len(payloads)
		}

		batch := Batch{
			ID:       len(batches) + 1,
			Payloads: payloads[i:end],
		}
		batches = append(batches, batch)
	}

	return batches
}

// GetBatchStats returns statistics about batch distribution
func (d *Distributor) GetBatchStats(batches []Batch) map[string]interface{} {
	if len(batches) == 0 {
		return map[string]interface{}{
			"totalBatches":   0,
			"totalPayloads":  0,
			"activateOrders": 0,
			"acceptedOrders": 0,
		}
	}

	totalPayloads := 0
	activateCount := 0
	acceptedCount := 0

	for _, batch := range batches {
		totalPayloads += len(batch.Payloads)
		for _, payload := range batch.Payloads {
			if payload.Type == TypeActivate {
				activateCount++
			} else {
				acceptedCount++
			}
		}
	}

	return map[string]interface{}{
		"totalBatches":   len(batches),
		"totalPayloads":  totalPayloads,
		"activateOrders": activateCount,
		"acceptedOrders": acceptedCount,
		"avgBatchSize":   float64(totalPayloads) / float64(len(batches)),
	}
}

// ValidateBatches ensures batches are properly formed
func ValidateBatches(batches []Batch) error {
	if len(batches) == 0 {
		return fmt.Errorf("no batches to validate")
	}

	for _, batch := range batches {
		if len(batch.Payloads) == 0 {
			return fmt.Errorf("batch %d has no payloads", batch.ID)
		}
	}

	return nil
}
