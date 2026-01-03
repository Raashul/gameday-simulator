package tests

import (
	"testing"

	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
)

func TestPayloadGeneration(t *testing.T) {
	cfg := &config.Config{
		Simulation: config.SimulationConfig{
			TotalOrders:    100,
			ActivatedCount: 70,
		},
		Payload: config.PayloadConfig{
			Location:          "US-EAST-1",
			POCOrder:          "POC-TEST-001",
			OrderNumberPrefix: "ORD-TEST-",
			CustomFields: map[string]interface{}{
				"priority": "high",
			},
		},
	}

	generator := payload.NewGenerator(cfg)
	payloads := generator.GenerateAll()

	// Test total count
	if len(payloads) != cfg.Simulation.TotalOrders {
		t.Errorf("Expected %d payloads, got %d", cfg.Simulation.TotalOrders, len(payloads))
	}

	// Count payload types
	activateCount := 0
	acceptedCount := 0

	for _, p := range payloads {
		if p.Type == payload.TypeActivate {
			activateCount++
		} else if p.Type == payload.TypeAccepted {
			acceptedCount++
		}

		// Verify required fields
		if p.OrderNumber == "" {
			t.Error("Order number should not be empty")
		}
		if p.Location != cfg.Payload.Location {
			t.Errorf("Expected location %s, got %s", cfg.Payload.Location, p.Location)
		}
	}

	// Verify type distribution
	if activateCount != cfg.Simulation.ActivatedCount {
		t.Errorf("Expected %d activate orders, got %d", cfg.Simulation.ActivatedCount, activateCount)
	}

	expectedAccepted := cfg.Simulation.TotalOrders - cfg.Simulation.ActivatedCount
	if acceptedCount != expectedAccepted {
		t.Errorf("Expected %d accepted orders, got %d", expectedAccepted, acceptedCount)
	}
}

func TestBatchDistribution(t *testing.T) {
	generator := payload.NewGenerator(&config.Config{
		Simulation: config.SimulationConfig{
			TotalOrders:    100,
			ActivatedCount: 70,
		},
		Payload: config.PayloadConfig{
			Location:          "US-EAST-1",
			POCOrder:          "POC-TEST-001",
			OrderNumberPrefix: "ORD-TEST-",
		},
	})

	payloads := generator.GenerateAll()

	distributor := payload.NewDistributor(20)
	batches := distributor.Distribute(payloads)

	// Test batch count (100 payloads / 20 per batch = 5 batches)
	expectedBatches := 5
	if len(batches) != expectedBatches {
		t.Errorf("Expected %d batches, got %d", expectedBatches, len(batches))
	}

	// Verify all payloads are distributed
	totalPayloads := 0
	for _, batch := range batches {
		totalPayloads += len(batch.Payloads)

		// Verify batch ID is set
		if batch.ID == 0 {
			t.Error("Batch ID should not be zero")
		}
	}

	if totalPayloads != len(payloads) {
		t.Errorf("Expected %d total payloads in batches, got %d", len(payloads), totalPayloads)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		shouldError bool
	}{
		{
			name: "Valid configuration",
			config: &config.Config{
				Simulation: config.SimulationConfig{
					TotalOrders:     100,
					BatchSize:       20,
					ParallelBatches: 5,
					ActivatedCount:  70,
				},
				API: config.APIConfig{
					BaseURL: "https://api.example.com",
					Timeout: 30,
				},
			},
			shouldError: false,
		},
		{
			name: "Invalid - activated count exceeds total",
			config: &config.Config{
				Simulation: config.SimulationConfig{
					TotalOrders:     100,
					BatchSize:       20,
					ParallelBatches: 5,
					ActivatedCount:  150,
				},
				API: config.APIConfig{
					BaseURL: "https://api.example.com",
					Timeout: 30,
				},
			},
			shouldError: true,
		},
		{
			name: "Invalid - missing base URL",
			config: &config.Config{
				Simulation: config.SimulationConfig{
					TotalOrders:     100,
					BatchSize:       20,
					ParallelBatches: 5,
					ActivatedCount:  70,
				},
				API: config.APIConfig{
					BaseURL: "",
					Timeout: 30,
				},
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Expected no validation error but got: %v", err)
			}
		})
	}
}
