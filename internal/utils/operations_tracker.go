package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// OperationsTracker tracks order IDs for cleanup purposes
type OperationsTracker struct {
	file      *os.File
	mu        sync.Mutex
	timestamp string
}

// NewOperationsTracker creates a new operations tracker with timestamp-based file
func NewOperationsTracker() (*OperationsTracker, error) {
	now := time.Now()

	// Create date-based directory: logs/2024-01-15
	dateDir := filepath.Join("logs", now.Format("2006-01-02"))
	if err := os.MkdirAll(dateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create operations directory: %w", err)
	}

	// Create timestamp for filename: 14-30-45
	timestamp := now.Format("15-04-05")
	fileName := fmt.Sprintf("operations_%s.txt", timestamp)
	filePath := filepath.Join(dateDir, fileName)

	// Open file for writing
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open operations file: %w", err)
	}

	return &OperationsTracker{
		file:      file,
		timestamp: timestamp,
	}, nil
}

// TrackOrder writes an order ID to the operations file
func (ot *OperationsTracker) TrackOrder(orderID string) error {
	ot.mu.Lock()
	defer ot.mu.Unlock()

	_, err := fmt.Fprintln(ot.file, orderID)
	if err != nil {
		return fmt.Errorf("failed to write order ID: %w", err)
	}

	// Flush to ensure data is written immediately
	return ot.file.Sync()
}

// GetTimestamp returns the timestamp used for this tracker
func (ot *OperationsTracker) GetTimestamp() string {
	return ot.timestamp
}

// Close closes the operations file
func (ot *OperationsTracker) Close() error {
	if ot.file != nil {
		return ot.file.Close()
	}
	return nil
}
