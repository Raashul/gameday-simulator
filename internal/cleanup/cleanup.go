package cleanup

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gameday-sim/internal/api"
	"gameday-sim/internal/utils"
)

// Cleaner handles cleanup of orphaned orders
type Cleaner struct {
	apiClient *api.Client
	logger    *utils.Logger
}

// NewCleaner creates a new cleanup handler
func NewCleaner(apiClient *api.Client, logger *utils.Logger) *Cleaner {
	return &Cleaner{
		apiClient: apiClient,
		logger:    logger,
	}
}

// CleanupByTimestamp reads operations file and cleans up orders
func (c *Cleaner) CleanupByTimestamp(ctx context.Context, timestamp string) error {
	// Find the operations file
	opsFilePath, err := findOperationsFile(timestamp)
	if err != nil {
		return fmt.Errorf("failed to find operations file: %w", err)
	}

	c.logger.Info("Starting cleanup", map[string]interface{}{
		"operationsFile": opsFilePath,
		"timestamp":      timestamp,
	})

	// Read order IDs from file
	orderIDs, err := readOrderIDs(opsFilePath)
	if err != nil {
		return fmt.Errorf("failed to read order IDs: %w", err)
	}

	c.logger.Info("Found orders to clean up", map[string]interface{}{
		"totalOrders": len(orderIDs),
	})

	// Process each order
	successCount := 0
	failedCount := 0

	for i, orderID := range orderIDs {
		c.logger.Info("Processing order", map[string]interface{}{
			"orderID":  orderID,
			"progress": fmt.Sprintf("%d/%d", i+1, len(orderIDs)),
		})

		if err := c.cleanupOrder(ctx, orderID); err != nil {
			c.logger.Error("Failed to cleanup order", map[string]interface{}{
				"orderID": orderID,
				"error":   err.Error(),
			})
			failedCount++
		} else {
			successCount++
		}
	}

	c.logger.Info("Cleanup complete", map[string]interface{}{
		"total":   len(orderIDs),
		"success": successCount,
		"failed":  failedCount,
	})

	return nil
}

// cleanupOrder handles cleanup for a single order
func (c *Cleaner) cleanupOrder(ctx context.Context, orderID string) error {
	// Get order details
	details, err := c.apiClient.GetDetails(ctx, orderID)
	if err != nil {
		return fmt.Errorf("failed to get order details: %w", err)
	}

	c.logger.Debug("Order details retrieved", map[string]interface{}{
		"orderID": orderID,
		"status":  details.Status,
	})

	// Determine action based on status
	if details.Status == "Accepted" {
		// Cancel accepted orders
		c.logger.Info("Cancelling accepted order", map[string]interface{}{
			"orderID": orderID,
		})
		_, err = c.apiClient.CancelOrder(ctx, orderID)
		if err != nil {
			return fmt.Errorf("failed to cancel order: %w", err)
		}
	} else {
		// End all other orders
		c.logger.Info("Ending order", map[string]interface{}{
			"orderID": orderID,
			"status":  details.Status,
		})
		_, err = c.apiClient.EndOrder(ctx, orderID)
		if err != nil {
			return fmt.Errorf("failed to end order: %w", err)
		}
	}

	return nil
}

// findOperationsFile searches for operations file by timestamp
func findOperationsFile(timestamp string) (string, error) {
	// Search in all date directories
	logsDir := "logs"
	entries, err := os.ReadDir(logsDir)
	if err != nil {
		return "", fmt.Errorf("failed to read logs directory: %w", err)
	}

	fileName := fmt.Sprintf("operations_%s.txt", timestamp)

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if operations file exists in this date directory
			filePath := filepath.Join(logsDir, entry.Name(), fileName)
			if _, err := os.Stat(filePath); err == nil {
				return filePath, nil
			}
		}
	}

	return "", fmt.Errorf("operations file not found for timestamp: %s", timestamp)
}

// readOrderIDs reads all order IDs from the operations file
func readOrderIDs(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var orderIDs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			orderIDs = append(orderIDs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return orderIDs, nil
}
