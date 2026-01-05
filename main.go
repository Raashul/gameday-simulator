package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gameday-sim/internal/api"
	"gameday-sim/internal/cleanup"
	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
	"gameday-sim/internal/reporter"
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

	// Check if cleanup mode is requested
	args := flag.Args()
	if len(args) > 0 && args[0] == "cleanup" {
		if len(args) < 2 {
			logger.Error("Cleanup mode requires timestamp argument", nil)
			fmt.Println("Usage: ./gameday-sim cleanup <timestamp>")
			fmt.Println("Example: ./gameday-sim cleanup 14-30-45")
			os.Exit(1)
		}
		runCleanupMode(args[1], logger)
		return
	}

	// Normal simulation mode
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

func runCleanupMode(timestamp string, logger *utils.Logger) {
	logger.Info("Starting cleanup mode", map[string]interface{}{
		"timestamp": timestamp,
	})

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize authentication
	logger.Info("Initializing authentication", nil)
	authManager := api.NewAuthManager(&cfg.OAuth, cfg.API.Timeout)
	_, err = authManager.GetToken(ctx)
	if err != nil {
		logger.Error("Failed to generate auth token", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	// Initialize API client
	apiClient := api.NewClient(cfg, authManager)

	// Run cleanup
	cleaner := cleanup.NewCleaner(apiClient, logger)
	if err := cleaner.CleanupByTimestamp(ctx, timestamp); err != nil {
		logger.Error("Cleanup failed", map[string]interface{}{
			"error": err.Error(),
		})
		os.Exit(1)
	}

	logger.Info("Cleanup completed successfully", nil)
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

	// Phase 5: Initialize operations tracker
	logger.Info("Phase 5: Initializing operations tracker", nil)
	opsTracker, err := utils.NewOperationsTracker()
	if err != nil {
		return fmt.Errorf("failed to create operations tracker: %w", err)
	}
	defer opsTracker.Close()
	logger.Info("Operations tracker created", map[string]interface{}{
		"timestamp": opsTracker.GetTimestamp(),
	})

	// Phase 6: Process batches
	logger.Info("Phase 6: Processing batches", map[string]interface{}{
		"parallelBatches": cfg.Simulation.ParallelBatches,
	})

	batchProcessor := simulator.NewBatchProcessor(apiClient, cfg, opsTracker)
	batchProcessor.StartTerminationWorker(ctx)

	//Start Batch Processing
	result, err := batchProcessor.ProcessBatches(ctx, batches)
	if err != nil {
		return fmt.Errorf("batch processing failed: %w", err)
	}

	//TODO: add cleaner reporting -> save to report folder with metrics
	// Phase 7: Report results
	logger.Info("Phase 7: Generating reports", nil)
	reporter.PrintResults(result, logger, time.Since(startTime))

	// Save detailed results to JSON
	// if err := reporter.SaveResultsToJSON(result, "simulation_results.json"); err != nil {
	// 	logger.Warn("Failed to save results to JSON", map[string]interface{}{
	// 		"error": err.Error(),
	// 	})
	// }

	return nil
}
