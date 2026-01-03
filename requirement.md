# Day in Life Simulator - Order Processing System Requirements
# Version: 2.0
# Language: Go (Golang)
# Type: API Load Testing & Simulation Tool with Geographical Path Generation

## PROJECT OVERVIEW
Build a Go-based simulation tool that mimics a day in the life of an order processing system. The simulator creates and manages 200+ orders through their lifecycle using 5 primary APIs, with configurable batch processing, parallel execution, and timing controls. Each order includes a unique GeoJSON polyline representing a geographical path, generated using a boundary-constrained zigzag crawl algorithm to ensure non-overlapping coverage within a specified area.

## CORE FUNCTIONAL REQUIREMENTS

### 1. API ENDPOINTS TO SIMULATE
The application must interact with the following 5 REST APIs:

1.1 CREATE ORDER API
- Method: POST /operation/payload
- Creates new orders asynchronously
- Returns: 202 Accepted with order ID
- Response includes initial order metadata

1.2 GET DETAILS API
- Method: GET /details
- Retrieves order status and details
- Called after configurable interval post-creation
- Must poll until status = "Accepted"
- Implements exponential backoff for retries

1.3 ACTIVATE ORDER API
- Method: POST /activate
- Activates eligible orders based on configuration
- Only called for orders marked for activation
- Returns activation confirmation

1.4 CANCEL ORDER API
- Method: POST /cancel
- Cancels non-activated orders after timeout
- Part of cleanup phase
- Returns cancellation confirmation

1.5 END ORDER API
- Method: POST /end
- Ends activated orders after specified duration
- Part of cleanup phase
- Returns completion status

### 2. PAYLOAD PRE-GENERATION REQUIREMENTS

2.1 PAYLOAD STRUCTURE
- Generate all order payloads BEFORE batch processing begins
- Stub payload template with configurable fields:
  ```go
  type OrderPayload struct {
      OrderNumber  string           `json:"orderNumber"`  // Unique identifier
      Location     string           `json:"location"`     // Static or from config
      POCOrder     string           `json:"pocOrder"`     // Point of Contact order
      Timestamp    time.Time        `json:"timestamp"`    // Generated at runtime
      Type         OrderType        `json:"type"`         // "accepted" or "activate"
      CustomFields map[string]interface{} `json:"customFields,omitempty"`
      Geometry     *GeoJSONGeometry `json:"geometry,omitempty"` // GeoJSON polyline
  }

  type GeoJSONGeometry struct {
      Type        string      `json:"type"`        // "LineString"
      Coordinates [][]float64 `json:"coordinates"` // [lng, lat] pairs
  }
  ```

2.2 PAYLOAD GENERATION LOGIC
- Pre-generate all N payloads at initialization
- Assign unique order numbers (e.g., "ORD-2024-000001" format)
- Randomly distribute order types based on activatedCount:
  - "activate" type: exactly activatedCount orders
  - "accepted" type: remaining orders (totalOrders - activatedCount)
- Generate GeoJSON polyline geometry for each payload:
  - Apply zigzag crawl pattern within boundary polygon
  - Ensure non-overlapping paths using delta spacing
  - Validate all coordinates are within boundary
- Optionally shuffle payload list for random distribution across batches
- Store payloads in memory for batch assignment

2.2.1 GEOGRAPHICAL PATH GENERATION
- **Base Polyline**: Start with a template polyline from config (3+ coordinate pairs)
- **Delta Spacing**: Apply configurable longitude/latitude offsets for each new order
- **Zigzag Pattern**: Crawl left-to-right on row 0, then down and right-to-left on row 1, etc.
- **Row Stacking**: Each row shifts down by (polyline_height + delta_latitude) for staircase effect
- **Boundary Constraint**: All polyline points must fall within configured polygon boundary
- **Point-in-Polygon Validation**: Use ray-casting algorithm to validate placement
- **Non-Overlapping**: Horizontal delta prevents same-row overlaps, vertical spacing prevents row overlaps

2.3 PAYLOAD ASSIGNMENT
- Divide pre-generated payloads into batches
- Each batch receives a slice of payloads
- Maintain payload-to-batch mapping for tracking

### 3. BATCH PROCESSING REQUIREMENTS

3.1 BATCH CONFIGURATION
- Support for 200+ total orders minimum
- Configurable batch sizes (e.g., 10, 20, 50 orders per batch)
- Define number of parallel batches

3.2 BATCH GOROUTINE INTERFACE
```go
type BatchProcessor interface {
    ProcessBatch(payloads []OrderPayload, batchType string) error
}

// Goroutine signature
func processBatch(
    batchID int,
    payloads []OrderPayload,
    orderType string, // "accepted" or "activate"
    config *Config,
    results chan<- BatchResult,
) {
    // Process each payload sequentially based on type
}
```

3.3 TYPE-BASED FLOW CONTROL
- Each batch goroutine receives:
  - List of pre-generated payloads
  - Order type determining the flow path
- Flow decision logic:
  - Type "activate": CREATE → GET (poll) → ACTIVATE → END (cleanup)
  - Type "accepted": CREATE → GET (poll) → CANCEL (cleanup)
- Batch maintains type consistency (all orders in batch have same type)

3.4 EXECUTION MODEL
- Batches execute in PARALLEL
- Items within each batch execute SEQUENTIALLY
- Implement proper goroutine management
- Use channels for coordination
- Implement worker pool pattern

3.5 TIMING CONTROLS
- Configurable wait intervals between API calls
- Support for different intervals per API type
- Implement jitter for realistic load distribution

### 4. ORDER LIFECYCLE MANAGEMENT

4.1 BUCKET PROCESSING PHASE
- Use pre-generated payload with timestamp
- Add any runtime fields to payload
- Execute flow based on order type:
  - "activate" type: Full activation flow
  - "accepted" type: Basic acceptance flow
- Wait configured interval after each creation
- Poll GET /details until status = "Accepted"
- Execute type-specific next steps

4.2 ASYNC TERMINATION PHASE
- Orders do NOT wait for termination (cancel/end) to complete
- After activation or acceptance timeout:
  - Push termination request to channel
  - ProcessOrder returns immediately (moves to next order)
  - Background worker processes termination requests asynchronously
- Termination Channel Architecture:
  ```go
  type TerminationRequest struct {
      OrderID string
      Action  TerminationAction // "cancel" or "end"
      Result  *OrderResult       // Pointer to update async
  }
  ```
- Background TerminationWorker goroutine:
  - Listens to termination channel
  - Executes cancel/end API calls
  - Updates OrderResult state asynchronously
  - Continues until context cancelled or channel closed

4.3 CLEANUP PHASE
- Track all created orders by type
- For "accepted" type orders:
  - Wait configured timeout, then schedule cancellation
  - Push to termination channel (async)
- For "activate" type orders:
  - Wait configured duration, then schedule end
  - Push to termination channel (async)
- Ensure all orders reach terminal state (eventually)
- Report cleanup completion statistics
- Order states include intermediate pending states:
  - `StatePendingCancel`: Queued for cancellation
  - `StatePendingEnd`: Queued for ending
  - `StateCancelled`: Cancellation complete
  - `StateEnded`: End operation complete

### 5. CONFIGURATION REQUIREMENTS

5.1 CONFIG.YAML STRUCTURE
```yaml
simulation:
  totalOrders: 200
  batchSize: 20
  parallelBatches: 5
  activatedCount: 170

payload:
  location: "US-EAST-1"
  pocOrder: "POC-2024-001"
  orderNumberPrefix: "ORD-2024-"
  customFields:
    priority: "normal"
    source: "simulator"
  basePolyline:
    coordinates:
      - [-96.79943798188481, 32.795102753983585]
      - [-96.79927289435462, 32.78885767285452]
      - [-96.79811728164334, 32.780252620552886]
  delta:
    longitude: 0.001  # Horizontal spacing between paths
    latitude: 0.001   # Additional vertical spacing between rows
  boundary:
    coordinates:      # GeoJSON Polygon format (first ring is exterior)
      - - [-96.80726593015929, 32.796582675082036]
        - [-96.80726593015929, 32.7781210082299]
        - [-96.78175523630775, 32.7781210082299]
        - [-96.78175523630775, 32.796582675082036]
        - [-96.80726593015929, 32.796582675082036]

intervals:
  betweenCreates: 2s
  afterCreateBeforeGet: 5s
  betweenGetPolls: 3s
  beforeActivate: 2s
  beforeCancel: 30s
  beforeEnd: 60s

api:
  baseUrl: "https://api.example.com"
  timeout: 30s
  retryMax: 3
  retryBackoff: 2s

cleanup:
  cancelTimeout: 300s
  endTimeout: 600s
  checkInterval: 10s
```

5.2 CONFIGURATION VALIDATION
- Validate all required fields present
- Ensure activatedCount <= totalOrders
- Validate time duration formats
- Check URL format validity
- Validate geographical constraints:
  - basePolyline must have at least 2 coordinate pairs
  - delta longitude and latitude must be positive
  - boundary polygon must have at least 3 points and be closed (first == last)
  - All coordinates in [longitude, latitude] format
  - Validate coordinate ranges: longitude [-180, 180], latitude [-90, 90]

### 6. TECHNICAL REQUIREMENTS

6.1 PROJECT STRUCTURE
```
day-in-life-simulator/
├── main.go
├── config.yaml
├── go.mod
├── go.sum
├── internal/
│   ├── api/
│   │   ├── client.go
│   │   ├── models.go
│   │   └── endpoints.go
│   ├── simulator/
│   │   ├── batch.go
│   │   ├── order.go
│   │   └── lifecycle.go
│   ├── payload/
│   │   ├── generator.go
│   │   ├── types.go
│   │   └── distributor.go
│   ├── config/
│   │   └── config.go
│   └── utils/
│       ├── logger.go
│       └── metrics.go
├── cmd/
│   └── simulate.go
└── tests/
    └── simulator_test.go
```

6.2 DEPENDENCIES
- Use standard library where possible
- HTTP client with retry logic (e.g., go-retryablehttp)
- YAML parser (gopkg.in/yaml.v3)
- Structured logging (e.g., zap or logrus)
- Metrics collection (prometheus client)
- CLI framework (cobra for command-line interface)

6.3 ERROR HANDLING
- Implement comprehensive error handling
- Retry failed API calls with exponential backoff
- Log all errors with context
- Graceful shutdown on SIGINT/SIGTERM
- Recovery from panics in goroutines

6.4 OBSERVABILITY
- Structured JSON logging
- Log levels (DEBUG, INFO, WARN, ERROR)
- Metrics for:
  - API response times
  - Success/failure rates per endpoint
  - Batch processing duration
  - Order state transitions
- Optional: Export metrics to Prometheus

### 7. PERFORMANCE REQUIREMENTS

7.1 CONCURRENCY
- Support minimum 10 parallel batches
- Efficient goroutine pooling
- Prevent resource exhaustion
- Implement rate limiting per API endpoint

7.2 RESOURCE MANAGEMENT
- Connection pooling for HTTP clients
- Configurable max connections
- Memory-efficient order tracking
- Cleanup of completed orders from memory

### 8. OUTPUT REQUIREMENTS

8.1 REPORTING
- Real-time progress updates
- Summary statistics upon completion:
  - Total orders created
  - Successful activations
  - Failed operations
  - Average response times
  - Total simulation duration

8.2 OUTPUT FORMATS
- Console output with progress bars
- JSON report file with detailed metrics
- CSV export of order lifecycle events
- Optional: HTML dashboard

8.3 GEOJSON VISUALIZATION
- Export generated paths to GeoJSON format for visualization
- Output directory: `logs/geojsons/`
- Filename format: `payloads_YYYYMMDD_HHMMSS.json`
- GeoJSON FeatureCollection includes:
  - Boundary polygon (red outline, semi-transparent fill)
  - Base polyline (blue, for reference)
  - All generated polylines (color-coded by order type):
    - Green: activate orders
    - Orange: accepted orders
  - Feature properties include: orderNumber, index, type
- Compatible with geojson.io for drag-and-drop visualization
- Enables visual verification of:
  - Non-overlapping paths
  - Boundary constraint compliance
  - Zigzag crawl pattern
  - Row stacking distribution

### 9. GEOGRAPHICAL PATH GENERATION REQUIREMENTS

9.1 COORDINATE SYSTEM
- Use WGS84 coordinate system (standard for GPS/GeoJSON)
- Coordinate format: [longitude, latitude] (GeoJSON standard)
- Longitude range: -180 to 180 degrees
- Latitude range: -90 to 90 degrees

9.2 ZIGZAG CRAWL ALGORITHM
- Start position: Top-left corner (row 0, column 0)
- Direction tracking:
  - Row 0: Move right (direction = +1)
  - Row 1: Move left (direction = -1)
  - Row 2: Move right (direction = +1)
  - Pattern alternates per row
- Column advancement:
  - Right direction: increment column counter
  - Left direction: decrement column counter
- Row transitions:
  - When path exceeds boundary, move to next row
  - Flip direction
  - Start from appropriate column (0 for right, maxCol for left)

9.3 STAIRCASE ROW STACKING
- Calculate polyline vertical extent: `height = max_lat - min_lat`
- Row spacing formula: `rowSpacing = polylineHeight + delta.latitude`
- Vertical offset per row: `offset = -rowSpacing * rowNumber`
- Ensures top of row N+1 is below bottom of row N
- Negative offset moves south (decreasing latitude)

9.4 BOUNDARY VALIDATION
- Point-in-Polygon algorithm: Ray casting method
- Validate ALL points of polyline are inside boundary
- If any point fails validation:
  - Reject current position
  - Advance to next row and flip direction
  - Retry with new position
- Continue until valid position found or boundary exhausted

9.5 COLLISION PREVENTION
- Horizontal spacing: delta.longitude between adjacent columns
- Vertical spacing: polylineHeight + delta.latitude between rows
- No two polylines share the same (row, column) position
- Zigzag pattern prevents vertical alignment overlaps
- Staircase ensures complete vertical separation

### 10. TESTING REQUIREMENTS

10.1 UNIT TESTS
- Test coverage minimum 80%
- Mock API responses
- Test batch processing logic
- Validate configuration parsing
- Test geographical algorithms:
  - Point-in-polygon validation
  - Zigzag crawl pattern
  - Row stacking calculations
  - Boundary constraint enforcement

10.2 INTEGRATION TESTS
- Test against mock server
- Validate order lifecycle transitions
- Test error recovery scenarios
- Test GeoJSON generation and validation

### 11. DOCUMENTATION REQUIREMENTS

11.1 CODE DOCUMENTATION
- GoDoc comments for all exported functions
- README.md with setup instructions
- Example configuration file
- API documentation
- Geographical path generation algorithm documentation

11.2 USER DOCUMENTATION
- Installation guide
- Configuration reference (including geographical fields)
- Troubleshooting guide
- Performance tuning tips
- GeoJSON visualization guide

### 12. ADDITIONAL FEATURES (NICE TO HAVE)

12.1 ADVANCED FEATURES
- Dry-run mode (simulate without API calls)
- Checkpoint/resume capability
- Multiple scenario support
- Load ramping (gradual increase)
- Circuit breaker pattern for APIs
- Distributed mode for multiple machines

12.2 MONITORING INTEGRATION
- Datadog APM integration
- New Relic support
- OpenTelemetry traces
- Custom webhooks for alerts

## IMPLEMENTATION NOTES FOR CLAUDE CODE

1. Start with payload pre-generation module before batch processing
2. Implement geographical path generation with boundary validation
3. Implement type-based routing in batch processor goroutines
4. Use interfaces for API client to enable easy mocking
5. Implement context.Context for cancellation propagation
6. Use sync.WaitGroup for batch coordination
7. Consider using errgroup for error handling in concurrent operations
8. Implement graceful shutdown with cleanup of in-flight requests
9. Use structured data types for order states and transitions
10. Consider implementing the State pattern for order lifecycle
11. Add comprehensive logging at each state transition
12. Build incrementally: start with payload generation, then single order, then batch, then parallel
13. Ensure payload distribution is deterministic for reproducible tests
14. Use channels for async termination phase (cancel/end operations)
15. Implement GeoJSON export for visual verification of path generation
16. Test geographical algorithms independently before integration

## ACCEPTANCE CRITERIA

1. Successfully create and process 200+ orders
2. Respect all configured timing intervals
3. Achieve specified activation ratio (e.g., 170/200)
4. All orders reach terminal state (ended or cancelled)
5. No goroutine leaks or race conditions
6. Graceful handling of API failures
7. Accurate reporting of simulation results
8. Configuration-driven behavior without code changes
9. All generated polylines fall within boundary polygon
10. No overlapping paths (verified via GeoJSON visualization)
11. Zigzag crawl pattern properly implemented
12. Async termination phase allows continuous order processing
13. GeoJSON export successfully generates valid FeatureCollection

## DELIVERABLES

1. Complete Go application source code
2. Comprehensive test suite (including geographical algorithm tests)
3. Sample config.yaml file with geographical configuration
4. README with setup and usage instructions
5. Dockerfile for containerized deployment
6. CI/CD pipeline configuration (GitHub Actions)
7. Performance benchmarking results
8. Sample GeoJSON output for visualization verification

## SUCCESS METRICS

- Zero data loss (all orders tracked)
- API error rate < 1%
- Memory usage < 500MB for 1000 orders
- Simulation accuracy within 5% of configured parameters
- Code maintainability score > B (using goreportcard)
- 100% of generated paths within boundary polygon
- Zero overlapping polylines

---

## VERSION HISTORY

### Version 2.0 (Current)
**Major Features Added:**
- GeoJSON polyline geometry generation for each order payload
- Configurable base polyline template with delta-based spacing
- Polygon boundary constraints with point-in-polygon validation
- Zigzag crawl pattern for non-overlapping path distribution
- Staircase row stacking algorithm for vertical separation
- Async termination phase using channels (non-blocking cancel/end)
- GeoJSON FeatureCollection export for visualization (geojson.io compatible)
- New order states: StatePendingCancel, StatePendingEnd
- Comprehensive geographical algorithm documentation

**Configuration Changes:**
- Added `basePolyline.coordinates` field
- Added `delta.longitude` and `delta.latitude` fields
- Added `boundary.coordinates` field (GeoJSON Polygon format)

**Architecture Changes:**
- Termination requests now processed asynchronously via channels
- Background TerminationWorker goroutine for cancel/end operations
- Generator tracks row/column position and direction for zigzag pattern
- Polyline height calculation for proper row spacing

### Version 1.0
Initial implementation with core order processing functionality.

---
END OF REQUIREMENTS