package payload

import "time"

// OrderType represents the type of order flow
type OrderType string

const (
	TypeActivate OrderType = "activate" // Orders that will be activated and ended
	TypeAccepted OrderType = "accepted" // Orders that will only be accepted and cancelled
)

// OrderPayload represents the structure of an order
type OrderPayload struct {
	OrderNumber  string                 `json:"orderNumber"`
	Location     string                 `json:"location"`
	POCOrder     string                 `json:"pocOrder"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         OrderType              `json:"type"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
	Geometry     *GeoJSONGeometry       `json:"geometry,omitempty"`
}

// GeoJSONGeometry represents a GeoJSON geometry (LineString)
type GeoJSONGeometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

// OrderState represents the current state of an order
type OrderState string

const (
	StateCreated       OrderState = "created"
	StateAccepted      OrderState = "accepted"
	StateActivated     OrderState = "activated"
	StatePendingCancel OrderState = "pending_cancel"
	StatePendingEnd    OrderState = "pending_end"
	StateCancelled     OrderState = "cancelled"
	StateEnded         OrderState = "ended"
	StateFailed        OrderState = "failed"
)

// TrackedOrder represents an order with its state and metadata
type TrackedOrder struct {
	Payload    OrderPayload
	OrderID    string
	State      OrderState
	CreatedAt  time.Time
	UpdatedAt  time.Time
	RetryCount int
	Error      error
}
