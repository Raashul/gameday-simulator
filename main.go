package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gameday-sim/internal/api"
	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
	"gameday-sim/internal/simulator"
	"gameday-sim/internal/utils"
)

var (
	configPath = flag.String("config", "config_dev.yaml", "Path to configuration file")
	logLevel   = flag.String("log-level", "INFO", "Log level (DEBUG, INFO, WARN, ERROR)")
)

func main() {
	flag.Parse()

	// Initialize logger
	logger := utils.NewLogger(utils.LogLevel(*logLevel))
	defer logger.Close()
	logger.Info("Starting Day-in-Life Simulator", nil)

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", map[string]interface{}{
			"error": err.Error(),
			"path":  *configPath,
		})
		os.Exit(1)
	}

	logger.Info("Configuration loaded successfully", map[string]interface{}{
		"totalOrders":     cfg.Simulation.TotalOrders,
		"batchSize":       cfg.Simulation.BatchSize,
		"parallelBatches": cfg.Simulation.ParallelBatches,
		"activatedCount":  cfg.Simulation.ActivatedCount,
	})

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", map[string]interface{}{
			"signal": sig.String(),
		})
		cancel()
	}()

	// Run simulation
	if err := runSimulation(ctx, cfg, logger); err != nil {
		if ctx.Err() != nil {
			logger.Info("Simulation cancelled", nil)
			os.Exit(0)
		}
		logger.Error("Simulation failed", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Simulation completed successfully", nil)
}

func runSimulation(ctx context.Context, cfg *config.Config, logger *utils.Logger) error {
	startTime := time.Now()

	// Phase 1: Generate payloads
	logger.Info("Phase 1: Generating payloads", nil)
	generator := payload.NewGenerator(cfg)
	payloads := generator.GenerateAll()
	generator.DumpGeoJSON(payloads)
	logger.Info("Payloads generated", map[string]interface{}{
		"totalPayloads": len(payloads),
	})

	//	Phase 2: Distribute into batches
	logger.Info("Phase 2: Distributing payloads into batches", nil)
	distributor := payload.NewDistributor(cfg.Simulation.BatchSize)
	batches := distributor.Distribute(payloads)

	if err := payload.ValidateBatches(batches); err != nil {
		return fmt.Errorf("batch validation failed: %w", err)
	}

	stats := distributor.GetBatchStats(batches)
	logger.Info("Batches created", stats)

	// Phase 3: Initialize authentication
	logger.Info("Phase 3: Initializing authentication", nil)
	authManager := api.NewAuthManager(&cfg.OAuth, cfg.API.Timeout)

	// Generate initial token
	logger.Info("Generating authentication token", nil)
	token, err := authManager.GetToken(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate auth token: %w", err)
	}
	logger.Info("Authentication token generated successfully", map[string]interface{}{
		"tokenLength": len(token),
	})

	// Phase 4: Initialize API client with authentication
	logger.Info("Phase 4: Initializing API client", nil)
	apiClient := api.NewClient(cfg, authManager)

	// Phase 5: Process batches
	logger.Info("Phase 5: Processing batches", map[string]interface{}{
		"parallelBatches": cfg.Simulation.ParallelBatches,
	})

	batchProcessor := simulator.NewBatchProcessor(apiClient, cfg)
	batchProcessor.StartTerminationWorker(ctx)
	result, err := batchProcessor.ProcessBatches(ctx, batches)
	if err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	// Phase 6: Report results
	logger.Info("Phase 6: Generating reports", nil)
	printResults(result, logger, time.Since(startTime))

	// Save detailed results to JSON
	if err := saveResultsToJSON(result, "simulation_results.json"); err != nil {
		logger.Warn("Failed to save results to JSON", map[string]interface{}{
			"error": err.Error(),
		})
	}

	return nil
}

func printResults(result *simulator.SimulationResult, logger *utils.Logger, totalDuration time.Duration) {
	stats := result.GetStats()

	separator := repeatString("=", 80)
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

func saveResultsToJSON(result *simulator.SimulationResult, filename string) error {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write results file: %w", err)
	}

	return nil
}

// repeatString repeats a string n times
func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}
