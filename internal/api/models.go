package api

import "time"

// CreateOrderRequest represents the request to create an order
type CreateOrderRequest struct {
	OrderNumber  string                 `json:"orderNumber"`
	Location     string                 `json:"location"`
	POCOrder     string                 `json:"pocOrder"`
	Timestamp    time.Time              `json:"timestamp"`
	Type         string                 `json:"type"`
	CustomFields map[string]interface{} `json:"customFields,omitempty"`
}

// CreateOrderResponse represents the response from create order API
type CreateOrderResponse struct {
	OrderID   string                 `json:"orderId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// GetDetailsResponse represents the response from get details API
type GetDetailsResponse struct {
	OrderID   string                 `json:"orderId"`
	Status    string                 `json:"status"` // "Pending", "Accepted", "Failed"
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ActivateOrderRequest represents the request to activate an order
type ActivateOrderRequest struct {
	OrderID string `json:"orderId"`
}

// ActivateOrderResponse represents the response from activate order API
type ActivateOrderResponse struct {
	OrderID   string    `json:"orderId"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// CancelOrderRequest represents the request to cancel an order
type CancelOrderRequest struct {
	OrderID string `json:"orderId"`
}

// CancelOrderResponse represents the response from cancel order API
type CancelOrderResponse struct {
	OrderID   string    `json:"orderId"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// EndOrderRequest represents the request to end an order
type EndOrderRequest struct {
	OrderID string `json:"orderId"`
}

// EndOrderResponse represents the response from end order API
type EndOrderResponse struct {
	OrderID   string    `json:"orderId"`
	Status    string    `json:"status"`
	Message   string    `json:"message,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error     string `json:"error"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}
