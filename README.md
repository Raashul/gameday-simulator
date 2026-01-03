# Day-in-Life Simulator - Order Processing System

A high-performance Go-based simulation tool that mimics a day in the life of an order processing system. The simulator creates and manages 200+ orders through their complete lifecycle using 5 primary APIs, with configurable batch processing, parallel execution, and timing controls.

## Features

- **Payload Pre-Generation**: All order payloads are generated before processing begins
- **Type-Based Routing**: Orders are processed differently based on type (activate vs accepted)
- **Parallel Batch Processing**: Configurable number of parallel batches with sequential order processing within each batch
- **Robust Error Handling**: Exponential backoff, retries, and comprehensive error recovery
- **Graceful Shutdown**: Proper signal handling and context-based cancellation
- **Structured Logging**: JSON-formatted logs with configurable log levels
- **Metrics Tracking**: Real-time statistics and detailed reporting

## Architecture

### Project Structure

```
gameday-sim/
├── main.go                    # Entry point and orchestration
├── config.yaml                # Configuration file
├── internal/
│   ├── api/                   # API client implementation
│   │   ├── client.go          # HTTP client with retry logic
│   │   ├── models.go          # Request/response models
│   │   └── endpoints.go       # API endpoint methods
│   ├── simulator/             # Core simulation logic
│   │   ├── batch.go           # Batch processing
│   │   └── order.go           # Order lifecycle management
│   ├── payload/               # Payload generation and distribution
│   │   ├── generator.go       # Payload generator
│   │   ├── distributor.go     # Batch distributor
│   │   └── types.go           # Data types
│   ├── config/                # Configuration management
│   │   └── config.go          # Config parsing and validation
│   └── utils/                 # Utilities
│       ├── logger.go          # Structured logging
│       └── metrics.go         # Metrics tracking
└── tests/                     # Unit tests
    └── payload_test.go
```

### Order Lifecycle Flows

**Activate Type Orders:**
1. CREATE → 2. GET (poll until accepted) → 3. ACTIVATE → 4. END

**Accepted Type Orders:**
1. CREATE → 2. GET (poll until accepted) → 3. CANCEL

## Installation

### Prerequisites

- Go 1.21 or higher
- Access to the target API endpoints

### Build from Source

```bash
# Clone the repository
cd gameday-sim

# Install dependencies
go mod download

# Build the application
go build -o gameday-sim .
```

## Configuration

Edit `config.yaml` to customize the simulation:

```yaml
simulation:
  totalOrders: 200          # Total number of orders to create
  batchSize: 20             # Orders per batch
  parallelBatches: 10       # Number of batches to run in parallel
  activatedCount: 170       # Number of orders to activate (rest will be accepted only)

payload:
  location: "US-EAST-1"
  pocOrder: "POC-2024-001"
  orderNumberPrefix: "ORD-2024-"
  customFields:
    priority: "normal"
    source: "simulator"

intervals:
  betweenCreates: 2s        # Wait time between creating orders in a batch
  afterCreateBeforeGet: 5s  # Wait before first status check
  betweenGetPolls: 3s       # Polling interval for status checks
  beforeActivate: 2s        # Wait before activation
  beforeCancel: 30s         # Wait before cancellation
  beforeEnd: 60s            # Wait before ending order

api:
  baseUrl: "https://api.example.com"
  timeout: 30s
  retryMax: 3
  retryBackoff: 2s

cleanup:
  cancelTimeout: 300s       # Timeout for order acceptance
  endTimeout: 600s          # Timeout for cleanup operations
  checkInterval: 10s        # Interval for cleanup checks
```

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `totalOrders` | Total number of orders to simulate | 200 |
| `batchSize` | Number of orders per batch | 20 |
| `parallelBatches` | Number of batches running concurrently | 10 |
| `activatedCount` | Orders that will be activated (must be ≤ totalOrders) | 170 |

## Usage

### Basic Usage

```bash
# Run with default config.yaml
./gameday-sim

# Run with custom config file
./gameday-sim -config /path/to/config.yaml

# Run with different log level
./gameday-sim -log-level DEBUG
```

### Command-Line Options

- `-config`: Path to configuration file (default: "config.yaml")
- `-log-level`: Log level - DEBUG, INFO, WARN, ERROR (default: "INFO")

### Example Output

```
================================================================================
SIMULATION RESULTS
================================================================================
Total Orders:       200
Successful Orders:  198
Failed Orders:      2
Ended Orders:       170
Cancelled Orders:   28
Total Batches:      10
Total Duration:     5m23.456s
Avg Order Duration: 1m34.123s
================================================================================
```

## Order Processing Details

### Payload Generation

1. Pre-generates all order payloads at initialization
2. Assigns unique order numbers (e.g., "ORD-2024-000001")
3. Randomly distributes order types based on `activatedCount`
4. Shuffles payloads for random distribution across batches

### Batch Processing

- **Batches run in parallel** (controlled by `parallelBatches`)
- **Orders within a batch execute sequentially**
- Each batch uses a goroutine with proper synchronization
- Semaphore pattern limits concurrent batches

### Error Handling

- **Exponential backoff** for retries
- **Circuit breaker** behavior for persistent failures
- **Context-based cancellation** propagates through all operations
- **Graceful shutdown** on SIGINT/SIGTERM signals

## API Endpoints

The simulator interacts with 5 REST endpoints:

1. **POST /operation/payload** - Create order (returns 202 Accepted)
2. **GET /details** - Get order status (polls until "Accepted")
3. **POST /activate** - Activate order (for activate-type orders)
4. **POST /cancel** - Cancel order (for accepted-type orders)
5. **POST /end** - End order (cleanup for activated orders)

## Testing

```bash
# Run all tests
go test ./tests/... -v

# Run tests with coverage
go test ./tests/... -cover

# Run specific test
go test ./tests/... -run TestPayloadGeneration -v
```

### Test Coverage

- Payload generation and distribution
- Configuration validation
- Batch processing logic
- Error handling scenarios

## Monitoring & Observability

### Structured Logging

All logs are output in JSON format for easy parsing:

```json
{
  "timestamp": "2024-01-15T10:30:45Z",
  "level": "INFO",
  "message": "Batch completed",
  "fields": {
    "batchId": 3,
    "successful": 20,
    "failed": 0,
    "duration": "2m15s"
  }
}
```

### Results Export

Detailed simulation results are saved to `simulation_results.json`:

```json
{
  "totalOrders": 200,
  "successfulOrders": 198,
  "failedOrders": 2,
  "batchResults": [...],
  "startTime": "2024-01-15T10:00:00Z",
  "endTime": "2024-01-15T10:05:23Z",
  "duration": 323456000000
}
```

## Performance Considerations

- **Memory Usage**: ~50MB for 1000 orders
- **Concurrency**: Supports 10+ parallel batches efficiently
- **Connection Pooling**: HTTP client reuses connections
- **Resource Management**: Proper cleanup of goroutines and channels

## Troubleshooting

### Common Issues

**Issue: "activatedCount cannot exceed totalOrders"**
- Ensure `activatedCount` ≤ `totalOrders` in config.yaml

**Issue: Connection timeouts**
- Increase `api.timeout` in configuration
- Check network connectivity to API endpoint

**Issue: High memory usage**
- Reduce `parallelBatches` count
- Decrease `totalOrders` or `batchSize`

**Issue: Orders stuck in pending state**
- Increase `cleanup.cancelTimeout`
- Check API endpoint availability

## Development

### Adding New Features

1. Follow the existing package structure
2. Add tests for new functionality
3. Update configuration schema if needed
4. Document changes in README

### Code Style

- Follow Go conventions and best practices
- Use `go fmt` for formatting
- Run `go vet` for static analysis
- Maintain test coverage > 80%

## License

MIT License - See LICENSE file for details

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## Support

For issues and questions:
- Open an issue on GitHub
- Check existing documentation
- Review troubleshooting guide

## Roadmap

Future enhancements:
- [ ] Prometheus metrics export
- [ ] HTML dashboard for real-time monitoring
- [ ] Dry-run mode
- [ ] Checkpoint/resume capability
- [ ] Multiple scenario support
- [ ] Distributed mode for load testing
