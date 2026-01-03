package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the complete application configuration
type Config struct {
	Simulation SimulationConfig `yaml:"simulation"`
	Payload    PayloadConfig    `yaml:"payload"`
	Intervals  IntervalConfig   `yaml:"intervals"`
	API        APIConfig        `yaml:"api"`
	Cleanup    CleanupConfig    `yaml:"cleanup"`
}

// SimulationConfig defines simulation parameters
type SimulationConfig struct {
	TotalOrders     int `yaml:"totalOrders"`
	BatchSize       int `yaml:"batchSize"`
	ParallelBatches int `yaml:"parallelBatches"`
	ActivatedCount  int `yaml:"activatedCount"`
}

// PayloadConfig defines payload generation settings
type PayloadConfig struct {
	Location          string                 `yaml:"location"`
	POCOrder          string                 `yaml:"pocOrder"`
	OrderNumberPrefix string                 `yaml:"orderNumberPrefix"`
	CustomFields      map[string]interface{} `yaml:"customFields"`
	BasePolyline      BasePolyline           `yaml:"basePolyline"`
	Delta             CoordinateDelta        `yaml:"delta"`
	Boundary          PolygonBoundary        `yaml:"boundary"`
}

// BasePolyline represents the base GeoJSON polyline coordinates
type BasePolyline struct {
	Coordinates [][]float64 `yaml:"coordinates"`
}

// CoordinateDelta represents the offset to apply for each new order
type CoordinateDelta struct {
	Longitude float64 `yaml:"longitude"`
	Latitude  float64 `yaml:"latitude"`
}

// PolygonBoundary represents the boundary polygon for volume generation (GeoJSON format)
type PolygonBoundary struct {
	Coordinates [][][]float64 `yaml:"coordinates"`
}

// IntervalConfig defines timing controls
type IntervalConfig struct {
	BetweenCreates       time.Duration `yaml:"betweenCreates"`
	AfterCreateBeforeGet time.Duration `yaml:"afterCreateBeforeGet"`
	BetweenGetPolls      time.Duration `yaml:"betweenGetPolls"`
	BeforeActivate       time.Duration `yaml:"beforeActivate"`
	BeforeCancel         time.Duration `yaml:"beforeCancel"`
	BeforeEnd            time.Duration `yaml:"beforeEnd"`
}

// APIConfig defines API client settings
type APIConfig struct {
	BaseURL      string        `yaml:"baseUrl"`
	Timeout      time.Duration `yaml:"timeout"`
	RetryMax     int           `yaml:"retryMax"`
	RetryBackoff time.Duration `yaml:"retryBackoff"`
}

// CleanupConfig defines cleanup phase settings
type CleanupConfig struct {
	CancelTimeout time.Duration `yaml:"cancelTimeout"`
	EndTimeout    time.Duration `yaml:"endTimeout"`
	CheckInterval time.Duration `yaml:"checkInterval"`
}

// Load reads and parses the configuration file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate ensures configuration is valid
func (c *Config) Validate() error {
	if c.Simulation.TotalOrders <= 0 {
		return fmt.Errorf("totalOrders must be positive")
	}

	if c.Simulation.BatchSize <= 0 {
		return fmt.Errorf("batchSize must be positive")
	}

	if c.Simulation.ParallelBatches <= 0 {
		return fmt.Errorf("parallelBatches must be positive")
	}

	if c.Simulation.ActivatedCount < 0 {
		return fmt.Errorf("activatedCount cannot be negative")
	}

	if c.Simulation.ActivatedCount > c.Simulation.TotalOrders {
		return fmt.Errorf("activatedCount (%d) cannot exceed totalOrders (%d)",
			c.Simulation.ActivatedCount, c.Simulation.TotalOrders)
	}

	if c.API.BaseURL == "" {
		return fmt.Errorf("API baseUrl is required")
	}

	if c.API.Timeout <= 0 {
		return fmt.Errorf("API timeout must be positive")
	}

	return nil
}
