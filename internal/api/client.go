package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gameday-sim/internal/config"
)

// Client represents the API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	retryMax   int
	backoff    time.Duration
}

// NewClient creates a new API client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL: cfg.API.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.API.Timeout,
		},
		retryMax: cfg.API.RetryMax,
		backoff:  cfg.API.RetryBackoff,
	}
}

// doRequest executes an HTTP request with retry logic
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}, target interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.retryMax; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			waitTime := c.backoff * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		err := c.executeRequest(ctx, method, path, body, target)
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't retry on client errors (4xx) except 429
		if httpErr, ok := err.(*HTTPError); ok {
			if httpErr.StatusCode >= 400 && httpErr.StatusCode < 500 && httpErr.StatusCode != 429 {
				return err
			}
		}
	}

	return fmt.Errorf("request failed after %d attempts: %w", c.retryMax+1, lastErr)
}

// executeRequest performs a single HTTP request
func (c *Client) executeRequest(ctx context.Context, method, path string, body interface{}, target interface{}) error {
	url := c.baseURL + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 400 {
		var errResp ErrorResponse
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return &HTTPError{
				StatusCode: resp.StatusCode,
				Message:    string(respBody),
			}
		}
		return &HTTPError{
			StatusCode: resp.StatusCode,
			Message:    errResp.Message,
			ErrorType:  errResp.Error,
		}
	}

	// Decode successful response
	if target != nil {
		if err := json.Unmarshal(respBody, target); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// HTTPError represents an HTTP error response
type HTTPError struct {
	StatusCode int
	Message    string
	ErrorType  string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP %d: %s - %s", e.StatusCode, e.ErrorType, e.Message)
}
