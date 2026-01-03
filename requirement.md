# Day in Life Simulator - Order Processing System Requirements
# Version: 1.0
# Language: Go (Golang)
# Type: API Load Testing & Simulation Tool

## PROJECT OVERVIEW
Build a Go-based simulation tool that mimics a day in the life of an order processing system. The simulator should create and manage 200+ orders through their lifecycle using 5 primary APIs, with configurable batch processing, parallel execution, and timing controls.

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
      OrderNumber  string    `json:"orderNumber"`  // Unique identifier
      Location     string    `json:"location"`     // Static or from config
      POCOrder     string    `json:"pocOrder"`     // Point of Contact order
      Timestamp    time.Time `json:"timestamp"`    // Generated at runtime
      Type         string    `json:"type"`         // "accepted" or "activate"
      CustomFields map[string]interface{} `json:"customFields,omitempty"`
  }
  ```

2.2 PAYLOAD GENERATION LOGIC
- Pre-generate all N payloads at initialization
- Assign unique order numbers (e.g., "ORD-2024-000001" format)
- Randomly distribute order types based on activatedCount:
  - "activate" type: exactly activatedCount orders
  - "accepted" type: remaining orders (totalOrders - activatedCount)
- Shuffle payload list for random distribution across batches
- Store payloads in memory for batch assignment

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

4.2 CLEANUP PHASE
- Track all created orders by type
- For "accepted" type orders:
  - Cancel after configured timeout
- For "activate" type orders:
  - End after configured duration
- Ensure all orders reach terminal state
- Report cleanup completion statistics

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

### 9. TESTING REQUIREMENTS

9.1 UNIT TESTS
- Test coverage minimum 80%
- Mock API responses
- Test batch processing logic
- Validate configuration parsing

9.2 INTEGRATION TESTS
- Test against mock server
- Validate order lifecycle transitions
- Test error recovery scenarios

### 10. DOCUMENTATION REQUIREMENTS

10.1 CODE DOCUMENTATION
- GoDoc comments for all exported functions
- README.md with setup instructions
- Example configuration file
- API documentation

10.2 USER DOCUMENTATION
- Installation guide
- Configuration reference
- Troubleshooting guide
- Performance tuning tips

### 11. ADDITIONAL FEATURES (NICE TO HAVE)

11.1 ADVANCED FEATURES
- Dry-run mode (simulate without API calls)
- Checkpoint/resume capability
- Multiple scenario support
- Load ramping (gradual increase)
- Circuit breaker pattern for APIs
- Distributed mode for multiple machines

11.2 MONITORING INTEGRATION
- Datadog APM integration
- New Relic support
- OpenTelemetry traces
- Custom webhooks for alerts

## IMPLEMENTATION NOTES FOR CLAUDE CODE

1. Start with payload pre-generation module before batch processing
2. Implement type-based routing in batch processor goroutines
3. Use interfaces for API client to enable easy mocking
4. Implement context.Context for cancellation propagation
5. Use sync.WaitGroup for batch coordination
6. Consider using errgroup for error handling in concurrent operations
7. Implement graceful shutdown with cleanup of in-flight requests
8. Use structured data types for order states and transitions
9. Consider implementing the State pattern for order lifecycle
10. Add comprehensive logging at each state transition
11. Build incrementally: start with payload generation, then single order, then batch, then parallel
12. Ensure payload distribution is deterministic for reproducible tests

## ACCEPTANCE CRITERIA

1. Successfully create and process 200+ orders
2. Respect all configured timing intervals
3. Achieve specified activation ratio (e.g., 170/200)
4. All orders reach terminal state (ended or cancelled)
5. No goroutine leaks or race conditions
6. Graceful handling of API failures
7. Accurate reporting of simulation results
8. Configuration-driven behavior without code changes

## DELIVERABLES

1. Complete Go application source code
2. Comprehensive test suite
3. Sample config.yaml file
4. README with setup and usage instructions
5. Dockerfile for containerized deployment
6. CI/CD pipeline configuration (GitHub Actions)
7. Performance benchmarking results

## SUCCESS METRICS

- Zero data loss (all orders tracked)
- API error rate < 1%
- Memory usage < 500MB for 1000 orders
- Simulation accuracy within 5% of configured parameters
- Code maintainability score > B (using goreportcard)

---
END OF REQUIREMENTS