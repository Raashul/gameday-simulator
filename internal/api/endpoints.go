package api

import (
	"context"
	"net/http"

	"gameday-sim/internal/payload"
)

// CreateOrder calls the create order API
func (c *Client) CreateOrder(ctx context.Context, payload payload.OrderPayload) (*CreateOrderResponse, error) {
	req := CreateOrderRequest{
		OrderNumber:  payload.OrderNumber,
		Location:     payload.Location,
		POCOrder:     payload.POCOrder,
		Timestamp:    payload.Timestamp,
		Type:         string(payload.Type),
		CustomFields: payload.CustomFields,
	}

	var resp CreateOrderResponse
	err := c.doRequest(ctx, http.MethodPost, "/operation/payload", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetDetails calls the get details API to check order status
func (c *Client) GetDetails(ctx context.Context, orderID string) (*GetDetailsResponse, error) {
	var resp GetDetailsResponse
	path := "/details?orderId=" + orderID
	err := c.doRequest(ctx, http.MethodGet, path, nil, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// ActivateOrder calls the activate order API
func (c *Client) ActivateOrder(ctx context.Context, orderID string) (*ActivateOrderResponse, error) {
	req := ActivateOrderRequest{
		OrderID: orderID,
	}

	var resp ActivateOrderResponse
	err := c.doRequest(ctx, http.MethodPost, "/activate", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// CancelOrder calls the cancel order API
func (c *Client) CancelOrder(ctx context.Context, orderID string) (*CancelOrderResponse, error) {
	req := CancelOrderRequest{
		OrderID: orderID,
	}

	var resp CancelOrderResponse
	err := c.doRequest(ctx, http.MethodPost, "/cancel", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// EndOrder calls the end order API
func (c *Client) EndOrder(ctx context.Context, orderID string) (*EndOrderResponse, error) {
	req := EndOrderRequest{
		OrderID: orderID,
	}

	var resp EndOrderResponse
	err := c.doRequest(ctx, http.MethodPost, "/end", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}
