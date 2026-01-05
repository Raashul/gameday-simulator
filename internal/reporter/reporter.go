package reporter

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gameday-sim/internal/simulator"
	"gameday-sim/internal/utils"
)

// PrintResults prints simulation results to console
func PrintResults(result *simulator.SimulationResult, logger *utils.Logger, totalDuration time.Duration) {
	stats := result.GetStats()

	separator := strings.Repeat("=", 80)
	fmt.Println("\n" + separator)
	fmt.Println("SIMULATION RESULTS")
	fmt.Println(separator)
	fmt.Printf("Total Orders:       %d\n", result.TotalOrders)
	fmt.Printf("Successful Orders:  %d\n", result.SuccessfulOrders)
	fmt.Printf("Failed Orders:      %d\n", result.FailedOrders)
	fmt.Printf("Ended Orders:       %v\n", stats["endedOrders"])
	fmt.Printf("Cancelled Orders:   %v\n", stats["cancelledOrders"])
	fmt.Printf("Total Batches:      %d\n", len(result.BatchResults))
	fmt.Printf("Total Duration:     %s\n", totalDuration.Round(time.Millisecond))
	fmt.Printf("Avg Order Duration: %v\n", stats["avgOrderDuration"])
	fmt.Println(separator)

	logger.Info("Simulation summary", stats)
}

// SaveResultsToJSON saves simulation results to a JSON file
func SaveResultsToJSON(result *simulator.SimulationResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	return nil
}
