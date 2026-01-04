package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"gameday-sim/internal/config"
)

// TokenResponse represents the OAuth token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
}

// AuthManager handles OAuth token generation and caching
type AuthManager struct {
	config      *config.OAuthConfig
	httpClient  *http.Client
	token       string
	tokenExpiry time.Time
	mu          sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(cfg *config.OAuthConfig, timeout time.Duration) *AuthManager {
	return &AuthManager{
		config: cfg,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetToken returns a valid token, generating a new one if needed
func (am *AuthManager) GetToken(ctx context.Context) (string, error) {
	am.mu.RLock()
	if am.token != "" && time.Now().Before(am.tokenExpiry) {
		token := am.token
		am.mu.RUnlock()
		return token, nil
	}
	am.mu.RUnlock()

	// Need to generate a new token
	return am.generateToken(ctx)
}

// generateToken requests a new OAuth token using password grant
func (am *AuthManager) generateToken(ctx context.Context) (string, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Double-check after acquiring write lock
	if am.token != "" && time.Now().Before(am.tokenExpiry) {
		return am.token, nil
	}

	// Prepare form data for password grant
	formData := url.Values{}
	formData.Set("grant_type", am.config.GrantType)
	formData.Set("username", am.config.Username)
	formData.Set("password", am.config.Password)
	formData.Set("client_id", am.config.ClientID)

	if am.config.ClientSecret != "" {
		formData.Set("client_secret", am.config.ClientSecret)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", am.config.TokenURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("token request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := parseJSONResponse(resp.Body, &tokenResp); err != nil {
		return "", fmt.Errorf("failed to parse token response: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("no access token in response")
	}

	// Cache the token
	am.token = tokenResp.AccessToken
	// Set expiry to slightly before actual expiry (5 minutes buffer)
	expiryDuration := time.Duration(tokenResp.ExpiresIn) * time.Second
	if expiryDuration > 5*time.Minute {
		expiryDuration -= 5 * time.Minute
	}
	am.tokenExpiry = time.Now().Add(expiryDuration)

	return am.token, nil
}

// ClearToken clears the cached token (useful for testing or forced refresh)
func (am *AuthManager) ClearToken() {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.token = ""
	am.tokenExpiry = time.Time{}
}
