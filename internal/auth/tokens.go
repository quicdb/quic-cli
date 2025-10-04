package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/quicdb/quic-cli/internal/config"
	"github.com/zalando/go-keyring"
)

const (
	service = "quic-cli"
	user    = "quic-user"
)

type TokenType string

const (
	AccessToken     TokenType = "access_token"
	RefreshToken    TokenType = "refresh_token"
	M2MClientID     TokenType = "m2m_client_id"
	M2MClientSecret TokenType = "m2m_client_secret"
)

// SaveToken persists the token securely in the OS keychain/credential manager
func SaveToken(token string, tokenType TokenType) error {
	return keyring.Set(service, user+string(tokenType), token)
}

// LoadToken retrieves the token from the OS keychain/credential manager
func LoadToken(tokenType TokenType) (string, error) {
	return keyring.Get(service, user+string(tokenType))
}

// DeleteToken removes the token from the OS keychain/credential manager
func DeleteToken(tokenType TokenType) error {
	return keyring.Delete(service, user+string(tokenType))
}

// ClearAllTokens removes all stored tokens (useful for logout)
func ClearAllTokens() error {
	// Try to delete all tokens and credentials, return the last error if any
	var lastErr error

	if err := DeleteToken(AccessToken); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		lastErr = err
	}

	if err := DeleteToken(RefreshToken); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		lastErr = err
	}

	if err := ClearM2MCredentials(); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		lastErr = err
	}

	return lastErr
}

// SaveM2MCredentials stores M2M client credentials securely
func SaveM2MCredentials(clientID, clientSecret string) error {
	if err := SaveToken(clientID, M2MClientID); err != nil {
		return fmt.Errorf("failed to save M2M client ID: %w", err)
	}
	if err := SaveToken(clientSecret, M2MClientSecret); err != nil {
		return fmt.Errorf("failed to save M2M client secret: %w", err)
	}
	return nil
}

// ClearM2MCredentials removes M2M credentials from keyring
func ClearM2MCredentials() error {
	var lastErr error

	if err := DeleteToken(M2MClientID); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		lastErr = err
	}

	if err := DeleteToken(M2MClientSecret); err != nil && !errors.Is(err, keyring.ErrNotFound) {
		lastErr = err
	}

	return lastErr
}

// RefreshTokenResponse represents the response from the token refresh
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	RequestID    string `json:"request_id"`
	StatusCode   int    `json:"status_code"`
}

// RefreshAccessToken exchanges a refresh token OR M2M credentials for a new access token
func RefreshAccessToken() error {
	// Try refresh token flow first (OAuth/PKCE)
	refreshToken, err := LoadToken(RefreshToken)
	if err == nil {
		return refreshWithRefreshToken(refreshToken)
	}

	// Fall back to M2M flow
	clientID, err := LoadToken(M2MClientID)
	if err != nil {
		return fmt.Errorf("no refresh token or M2M credentials found: %w", err)
	}

	clientSecret, err := LoadToken(M2MClientSecret)
	if err != nil {
		return fmt.Errorf("M2M client ID found but secret missing: %w", err)
	}

	return refreshWithM2MCredentials(clientID, clientSecret)
}

// refreshWithRefreshToken uses OAuth refresh token to get new access token
func refreshWithRefreshToken(refreshToken string) error {
	cfg := config.Get()
	url := fmt.Sprintf("%s/v1/public/%s/oauth2/token", cfg.StytchURL, cfg.ProjectID)

	// Prepare the request body
	body := map[string]string{
		"client_id":     cfg.ClientID,
		"refresh_token": refreshToken,
		"grant_type":    "refresh_token",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making refresh request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading refresh response: %w", err)
	}

	// Parse the response
	var tokenResp RefreshTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return fmt.Errorf("error parsing refresh response: %w", err)
	}

	if tokenResp.StatusCode != 200 {
		return fmt.Errorf("refresh token error: %s", string(respBody))
	}

	// Save the new tokens
	if err := SaveToken(tokenResp.AccessToken, AccessToken); err != nil {
		return fmt.Errorf("failed to save new access token: %w", err)
	}

	// Update refresh token if a new one was provided
	if tokenResp.RefreshToken != "" {
		if err := SaveToken(tokenResp.RefreshToken, RefreshToken); err != nil {
			return fmt.Errorf("failed to save new refresh token: %w", err)
		}
	}

	return nil
}

// refreshWithM2MCredentials uses M2M client credentials to get new access token
func refreshWithM2MCredentials(clientID, clientSecret string) error {
	cfg := config.Get()
	url := fmt.Sprintf("%s/v1/public/%s/oauth2/token", cfg.StytchURL, cfg.ProjectID)

	// Prepare the request body for client credentials grant
	body := map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     clientID,
		"client_secret": clientSecret,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("error marshaling request body: %w", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making M2M token request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading M2M token response: %w", err)
	}

	// Parse the response
	var tokenResp RefreshTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return fmt.Errorf("error parsing M2M token response: %w", err)
	}

	if tokenResp.StatusCode != 200 {
		return fmt.Errorf("M2M token error: %s", string(respBody))
	}

	// Save the new access token
	if err := SaveToken(tokenResp.AccessToken, AccessToken); err != nil {
		return fmt.Errorf("failed to save new access token: %w", err)
	}

	return nil
}
