# Day-in-Life Simulator - Order Processing System

A high-performance Go-based simulation tool that mimics a day in the life of an order processing system. The simulator creates and manages 200+ orders through their complete lifecycle using 5 primary APIs, with configurable batch processing, parallel execution, and timing controls. Each order includes a unique GeoJSON polyline representing a geographical path, generated using a boundary-constrained zigzag crawl algorithm.

## Features

- **Payload Pre-Generation**: All order payloads are generated before processing begins
- **Geographical Path Generation**: Each order includes a unique GeoJSON polyline with boundary constraints
- **Zigzag Crawl Pattern**: Non-overlapping paths distributed using intelligent staircase algorithm
- **Boundary Validation**: Point-in-polygon algorithm ensures all paths stay within configured area
- **Type-Based Routing**: Orders are processed differently based on type (activate vs accepted)
- **Async Termination**: Non-blocking cancel/end operations using channel-based architecture
- **Parallel Batch Processing**: Configurable number of parallel batches with sequential order processing within each batch
- **GeoJSON Export**: Visual verification via geojson.io compatible output files
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
1. CREATE → 2. GET (poll until accepted) → 3. ACTIVATE → 4. Schedule END (async) → ProcessOrder returns
   - Background worker processes END operation asynchronously

**Accepted Type Orders:**
1. CREATE → 2. GET (poll until accepted) → 2. Schedule CANCEL (async) → ProcessOrder returns
   - Background worker processes CANCEL operation asynchronously

**Order States:**
- `StateCreated` → `StateAccepted` → `StateActivated` → `StatePendingEnd` → `StateEnded`
- `StateCreated` → `StateAccepted` → `StatePendingCancel` → `StateCancelled`
- `StateFailed` (on any error)

## Quick Start

```bash
# 1. Build the application
go build -o gameday-sim .

# 2. Edit config.yaml with your settings
#    - Set API endpoints
#    - Configure geographical boundary
#    - Adjust timing parameters

# 3. Run the simulator
./gameday-sim

# 4. View generated paths
# Open logs/geojsons/payloads_*.json in geojson.io
```

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
  basePolyline:              # Template polyline for path generation
    coordinates:
      - [-96.79943798188481, 32.795102753983585]
      - [-96.79927289435462, 32.78885767285452]
      - [-96.79811728164334, 32.780252620552886]
  delta:                     # Spacing between paths
    longitude: 0.001         # Horizontal spacing (east-west)
    latitude: 0.001          # Additional vertical spacing between rows
  boundary:                  # Polygon boundary (GeoJSON format)
    coordinates:
      - - [-96.80726593015929, 32.796582675082036]
        - [-96.80726593015929, 32.7781210082299]
        - [-96.78175523630775, 32.7781210082299]
        - [-96.78175523630775, 32.796582675082036]
        - [-96.80726593015929, 32.796582675082036]

intervals:
  betweenCreates: 2s        # Wait time between creating orders in a batch
  afterCreateBeforeGet: 5s  # Wait before first status check
  betweenGetPolls: 3s       # Polling interval for status checks
  beforeActivate: 2s        # Wait before activation
  beforeCancel: 30s         # Wait before scheduling cancellation
  beforeEnd: 60s            # Wait before scheduling end operation

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

#### Simulation Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `totalOrders` | Total number of orders to simulate | 200 |
| `batchSize` | Number of orders per batch | 20 |
| `parallelBatches` | Number of batches running concurrently | 10 |
| `activatedCount` | Orders that will be activated (must be ≤ totalOrders) | 170 |

#### Geographical Parameters

| Parameter | Description | Format |
|-----------|-------------|--------|
| `basePolyline.coordinates` | Template polyline for path generation | Array of [lng, lat] pairs |
| `delta.longitude` | Horizontal spacing between paths (degrees) | Float (e.g., 0.001) |
| `delta.latitude` | Additional vertical spacing between rows (degrees) | Float (e.g., 0.001) |
| `boundary.coordinates` | Polygon boundary constraint (GeoJSON Polygon) | Array of rings, each ring is array of [lng, lat] |

**Geographical Behavior:**
- Paths are generated in a **zigzag pattern**: left-to-right on row 0, right-to-left on row 1, etc.
- Each row is stacked **vertically** with spacing = (polyline_height + delta.latitude)
- All polyline points must fall **within the boundary** polygon
- Point-in-polygon validation uses **ray-casting algorithm**

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
Activated Orders:   170
Ended Orders:       165
Cancelled Orders:   28
Pending End:        5
Pending Cancel:     0
Total Batches:      10
Total Duration:     5m23.456s
Avg Order Duration: 1m34.123s
GeoJSON Output:     logs/geojsons/payloads_20260103_162230.json
================================================================================
```

## Order Processing Details

### Payload Generation

1. Pre-generates all order payloads at initialization
2. Assigns unique order numbers (e.g., "ORD-2024-000001")
3. Generates unique GeoJSON polyline for each order:
   - Applies zigzag crawl pattern within boundary polygon
   - Ensures non-overlapping paths using delta spacing
   - Validates all coordinates are within boundary
4. Distributes order types based on `activatedCount`
5. Optionally shuffles payloads for random distribution across batches

### Batch Processing

- **Batches run in parallel** (controlled by `parallelBatches`)
- **Orders within a batch execute sequentially**
- Each batch uses a goroutine with proper synchronization
- Semaphore pattern limits concurrent batches

### Geographical Path Generation

The simulator uses an intelligent algorithm to generate unique, non-overlapping paths:

**Zigzag Crawl Pattern:**
```
Row 0:  →→→→→ (moving right, incrementing column)
Row 1:  ←←←←← (moving left, decrementing column)
Row 2:  →→→→→ (moving right again)
```

**Staircase Stacking:**
- Each row is offset vertically by: `polylineHeight + delta.latitude`
- Ensures the top of row N+1 is below the bottom of row N
- Creates a "staircase" effect with no vertical overlap

**Boundary Validation:**
- Uses ray-casting point-in-polygon algorithm
- Tests every coordinate of every polyline
- Rejects positions that exceed boundary
- Automatically moves to next row when out of space

**Collision Prevention:**
- Horizontal: `delta.longitude` spacing between columns
- Vertical: `polylineHeight + delta.latitude` spacing between rows
- Zigzag pattern prevents vertical alignment
- Each (row, column) position is unique

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
- GeoJSON polyline generation
- Point-in-polygon validation
- Zigzag crawl pattern logic
- Boundary constraint enforcement
- Configuration validation
- Batch processing logic
- Error handling scenarios
- Async termination channel operations

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

### GeoJSON Visualization

Generated paths are automatically exported to `logs/geojsons/payloads_YYYYMMDD_HHMMSS.json` for visual verification:

**To Visualize:**
1. Navigate to https://geojson.io
2. Drag and drop the generated JSON file
3. View the map showing:
   - **Red boundary** polygon with semi-transparent fill
   - **Blue base polyline** (reference point at row 0, col 0)
   - **Green paths** for activate-type orders
   - **Orange paths** for accepted-type orders

**What to Verify:**
- ✓ All paths stay within the red boundary
- ✓ No overlapping polylines
- ✓ Zigzag pattern visible (alternating row directions)
- ✓ Staircase stacking (each row below the previous)

Click on any path to see properties: `orderNumber`, `index`, `type`

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

**Issue: "All paths exceed boundary" or insufficient paths generated**
- Verify `boundary.coordinates` polygon is large enough
- Check that `basePolyline` starts within the boundary (top-left corner)
- Increase boundary size or decrease `delta` values
- Reduce `totalOrders` to fit within available space

**Issue: Overlapping paths in GeoJSON visualization**
- Increase `delta.longitude` for more horizontal spacing
- Increase `delta.latitude` for more vertical row spacing
- Verify generator is not shuffling payloads (check shuffle is disabled)

**Issue: Invalid GeoJSON format errors**
- Ensure all coordinates are in [longitude, latitude] format (not lat, lng)
- Verify longitude range: -180 to 180, latitude range: -90 to 90
- Check boundary polygon is properly closed (first point equals last point)

## Geographical Configuration Best Practices

### Choosing Delta Values

**Rule of thumb for spacing:**
- `delta.longitude = 0.001` ≈ 111 meters east-west at equator
- `delta.latitude = 0.001` ≈ 111 meters north-south
- Adjust based on your polyline width and desired spacing
- Larger delta = fewer paths fit in boundary, more spacing

### Designing Boundaries

**Tips for boundary polygons:**
1. Start with a rectangular boundary for simplicity
2. Place `basePolyline` at the top-left corner of boundary
3. Ensure boundary is large enough for desired `totalOrders`
4. Use geojson.io to draw and test your boundary first
5. Copy coordinates from geojson.io directly into config.yaml

**Estimating capacity:**
```
Columns per row ≈ (boundary_width / delta.longitude)
Rows available ≈ (boundary_height / (polyline_height + delta.latitude))
Max orders ≈ columns × rows
```

### Example: Small Test Area (Dallas, TX)

```yaml
basePolyline:
  coordinates:  # Small 3-point route
    - [-96.7994, 32.7951]
    - [-96.7993, 32.7889]
    - [-96.7981, 32.7803]
delta:
  longitude: 0.0005  # ~55m spacing
  latitude: 0.0005   # ~55m spacing
boundary:
  coordinates:  # ~2km × 2km area
    - - [-96.81, 32.80]
      - [-96.78, 32.80]
      - [-96.78, 32.77]
      - [-96.81, 32.77]
      - [-96.81, 32.80]
```

This configuration supports ~2,000+ non-overlapping paths.

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

### Completed (v2.0)
- [x] GeoJSON polyline generation with boundary constraints
- [x] Zigzag crawl pattern for non-overlapping paths
- [x] Point-in-polygon validation
- [x] Async termination phase with channels
- [x] GeoJSON export for visual verification

### Future Enhancements
- [ ] Prometheus metrics export
- [ ] HTML dashboard for real-time monitoring
- [ ] Dry-run mode
- [ ] Checkpoint/resume capability
- [ ] Multiple scenario support
- [ ] Distributed mode for load testing
- [ ] Custom path generation algorithms (spiral, grid, random)
- [ ] Multi-polygon boundary support
- [ ] Real-time GeoJSON streaming
