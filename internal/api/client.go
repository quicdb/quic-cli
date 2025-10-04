package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/quicdb/quic-cli/internal/config"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return e.Message
}

type CreateBranchRequest struct {
	Name string `json:"name"`
}

type CreateBranchResponse struct {
	User     string `json:"user"`
	Password string `json:"password"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

type Instance struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Subdomain        string `json:"subdomain"`
	Region           string `json:"region"`
	SelectedDatabase string `json:"selected_database"`
}

type Branch struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Cluster   string `json:"cluster"`    // Instance.Name
	CreatedBy string `json:"created_by"` // CreatedByID
	CreatedAt string `json:"created_at"`
}

func NewClient() *Client {
	cfg := config.Get()

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second, // Default timeout
		},
		baseURL: cfg.APIURL,
	}
}

func (c *Client) CreateBranch(ctx context.Context, instanceID, branchName string, timeout time.Duration) (*CreateBranchResponse, error) {
	// Prepare request body
	reqBody := CreateBranchRequest{
		Name: branchName,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	// Create request
	url := fmt.Sprintf("%s/instances/%s/branches", c.baseURL, instanceID)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create a client with the specified timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Make authenticated request with retry
	body, err := c.makeAuthenticatedRequest(client, req)
	if err != nil {
		return nil, err
	}

	// Parse successful response
	var branchResp CreateBranchResponse
	if err := json.Unmarshal(body, &branchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &branchResp, nil
}

func (c *Client) DeleteBranch(ctx context.Context, instanceID, branchName string) error {
	// Create request
	url := fmt.Sprintf("%s/instances/%s/branches/%s", c.baseURL, instanceID, branchName)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Make authenticated request with retry
	body, err := c.makeAuthenticatedRequest(c.httpClient, req)
	if err != nil {
		return err
	}

	// Parse response to verify success
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	return nil
}

func (c *Client) ListInstances(ctx context.Context) ([]Instance, error) {
	// Create request
	url := fmt.Sprintf("%s/instances", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make authenticated request with retry
	body, err := c.makeAuthenticatedRequest(c.httpClient, req)
	if err != nil {
		return nil, err
	}

	// Parse successful response
	var instances []Instance
	if err := json.Unmarshal(body, &instances); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return instances, nil
}

func (c *Client) ListBranches(ctx context.Context) ([]Branch, error) {
	// Create request
	url := fmt.Sprintf("%s/branches", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Make authenticated request with retry
	body, err := c.makeAuthenticatedRequest(c.httpClient, req)
	if err != nil {
		return nil, err
	}

	// Parse successful response
	var branches []Branch
	if err := json.Unmarshal(body, &branches); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return branches, nil
}

// makeAuthenticatedRequest handles authentication with automatic token refresh
func (c *Client) makeAuthenticatedRequest(client *http.Client, req *http.Request) ([]byte, error) {
	// Capture request body before making any requests (since body can only be read once)
	var reqBody []byte
	if req.Body != nil {
		var err error
		reqBody, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body.Close()
		req.Body = io.NopCloser(bytes.NewBuffer(reqBody))
	}

	// Try with current access token first
	token, err := auth.LoadToken(auth.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("no valid authentication token found. Please run 'quic login' first")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// If we get 401 Unauthorized, try to refresh the token and retry once
	if resp.StatusCode == 401 {
		// Attempt to refresh the access token
		if refreshErr := auth.RefreshAccessToken(); refreshErr != nil {
			return nil, fmt.Errorf("authentication failed and token refresh failed: %w", refreshErr)
		}

		// Get the new access token
		newToken, err := auth.LoadToken(auth.AccessToken)
		if err != nil {
			return nil, fmt.Errorf("failed to load refreshed token: %w", err)
		}

		// Create retry request with saved body
		retryReq, err := http.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), bytes.NewBuffer(reqBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create retry request: %w", err)
		}

		// Copy headers and update authorization
		for key, values := range req.Header {
			for _, value := range values {
				retryReq.Header.Add(key, value)
			}
		}
		retryReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", newToken))

		// Send retry request
		retryResp, err := client.Do(retryReq)
		if err != nil {
			return nil, fmt.Errorf("failed to make retry request: %w", err)
		}
		defer retryResp.Body.Close()

		// Read retry response body
		body, err = io.ReadAll(retryResp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read retry response body: %w", err)
		}

		resp = retryResp
	}

	// Handle error responses
	if resp.StatusCode >= 400 {
		var errorResp map[string]interface{}
		if err := json.Unmarshal(body, &errorResp); err == nil {
			if msg, ok := errorResp["error"].(string); ok {
				return nil, &APIError{
					StatusCode: resp.StatusCode,
					Message:    msg,
				}
			}
		}
		return nil, fmt.Errorf("API error: HTTP %d - %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// M2MTokenResponse represents the response from M2M token exchange
type M2MTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
}

// ExchangeM2MToken exchanges M2M client credentials for access tokens
func ExchangeM2MToken(clientID, clientSecret, stytchURL, projectID string) (*M2MTokenResponse, error) {
	tokenURL := fmt.Sprintf("%s/v1/public/%s/oauth2/token", stytchURL, projectID)

	// Prepare the request body for client credentials grant
	body := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     clientID,
		"client_secret": clientSecret,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", tokenURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Parse the response
	var tokenResp M2MTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("error from API (status %d): %s", resp.StatusCode, string(respBody))
	}

	return &tokenResp, nil
}
