package simulator

import (
	"context"
	"fmt"
	"time"

	"gameday-sim/internal/api"
	"gameday-sim/internal/config"
	"gameday-sim/internal/payload"
)

// TerminationRequest represents an order that needs to be terminated
type TerminationRequest struct {
	OrderID string
	Action  TerminationAction
	Result  *OrderResult
}

// TerminationAction defines the type of termination
type TerminationAction string

const (
	ActionEnd    TerminationAction = "end"
	ActionCancel TerminationAction = "cancel"
)

// OrderProcessor handles individual order lifecycle
type OrderProcessor struct {
	apiClient       *api.Client
	config          *config.Config
	terminationChan chan<- TerminationRequest
}

// NewOrderProcessor creates a new order processor
func NewOrderProcessor(apiClient *api.Client, cfg *config.Config, terminationChan chan<- TerminationRequest) *OrderProcessor {
	return &OrderProcessor{
		apiClient:       apiClient,
		config:          cfg,
		terminationChan: terminationChan,
	}
}

// ProcessOrder executes the full lifecycle for an order based on its type
func (p *OrderProcessor) ProcessOrder(ctx context.Context, pl payload.OrderPayload) (*OrderResult, error) {
	result := &OrderResult{
		OrderNumber: pl.OrderNumber,
		Type:        pl.Type,
		StartTime:   time.Now(),
	}

	// Step 1: Create the order
	createResp, err := p.createOrder(ctx, pl)
	if err != nil {
		result.Error = err
		result.State = payload.StateFailed
		return result, err
	}

	result.OrderID = createResp.OrderID
	result.State = payload.StateCreated

	// Step 2: Wait and poll for acceptance
	if err := p.waitForAcceptance(ctx, createResp.OrderID); err != nil {
		result.Error = err
		result.State = payload.StateFailed
		return result, err
	}

	result.State = payload.StateAccepted

	// Step 3: Execute type-specific flow
	if pl.Type == payload.TypeActivate {
		if err := p.activateFlow(ctx, createResp.OrderID, result); err != nil {
			result.Error = err
			return result, err
		}
	} else {
		if err := p.acceptedFlow(ctx, createResp.OrderID, result); err != nil {
			result.Error = err
			return result, err
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// createOrder creates a new order via API
func (p *OrderProcessor) createOrder(ctx context.Context, pl payload.OrderPayload) (*api.CreateOrderResponse, error) {
	resp, err := p.apiClient.CreateOrder(ctx, pl)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	return resp, nil
}

// waitForAcceptance polls the details API until order is accepted
func (p *OrderProcessor) waitForAcceptance(ctx context.Context, orderID string) error {
	// Wait initial interval after creation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.config.Intervals.AfterCreateBeforeGet):
	}

	// Poll until accepted with timeout
	timeout := time.After(p.config.Cleanup.CancelTimeout)
	ticker := time.NewTicker(p.config.Intervals.BetweenGetPolls)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for order acceptance")
		case <-ticker.C:
			resp, err := p.apiClient.GetDetails(ctx, orderID)
			if err != nil {
				return fmt.Errorf("failed to get order details: %w", err)
			}

			if resp.Status == "Accepted" {
				return nil
			}

			if resp.Status == "Failed" {
				return fmt.Errorf("order failed during processing")
			}
		}
	}
}

// activateFlow handles the activation flow: activate -> schedule end
func (p *OrderProcessor) activateFlow(ctx context.Context, orderID string, result *OrderResult) error {
	// Wait before activation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.config.Intervals.BeforeActivate):
	}

	// Activate the order
	_, err := p.apiClient.ActivateOrder(ctx, orderID)
	if err != nil {
		result.State = payload.StateFailed
		return fmt.Errorf("failed to activate order: %w", err)
	}

	result.State = payload.StateActivated

	// Wait before scheduling termination
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.config.Intervals.BeforeEnd):
	}

	// Push to termination channel for async processing
	p.terminationChan <- TerminationRequest{
		OrderID: orderID,
		Action:  ActionEnd,
		Result:  result,
	}

	result.State = payload.StatePendingEnd
	return nil
}

// acceptedFlow handles the accepted-only flow: schedule cancel
func (p *OrderProcessor) acceptedFlow(ctx context.Context, orderID string, result *OrderResult) error {
	// Wait before scheduling cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(p.config.Intervals.BeforeCancel):
	}

	// Push to termination channel for async processing
	p.terminationChan <- TerminationRequest{
		OrderID: orderID,
		Action:  ActionCancel,
		Result:  result,
	}

	result.State = payload.StatePendingCancel
	return nil
}

// OrderResult represents the result of processing an order
type OrderResult struct {
	OrderNumber string
	OrderID     string
	Type        payload.OrderType
	State       payload.OrderState
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Error       error
}

// TerminationWorker processes termination requests from the channel
func TerminationWorker(ctx context.Context, apiClient *api.Client, terminationChan <-chan TerminationRequest) {
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-terminationChan:
			processTermination(ctx, apiClient, req)
		}
	}
}

// processTermination handles the actual termination API call
func processTermination(ctx context.Context, apiClient *api.Client, req TerminationRequest) {
	switch req.Action {
	case ActionEnd:
		_, err := apiClient.EndOrder(ctx, req.OrderID)
		if err != nil {
			req.Result.State = payload.StateFailed
			req.Result.Error = fmt.Errorf("failed to end order: %w", err)
		} else {
			req.Result.State = payload.StateEnded
		}
	case ActionCancel:
		_, err := apiClient.CancelOrder(ctx, req.OrderID)
		if err != nil {
			req.Result.State = payload.StateFailed
			req.Result.Error = fmt.Errorf("failed to cancel order: %w", err)
		} else {
			req.Result.State = payload.StateCancelled
		}
	}

	req.Result.EndTime = time.Now()
	req.Result.Duration = req.Result.EndTime.Sub(req.Result.StartTime)
}
