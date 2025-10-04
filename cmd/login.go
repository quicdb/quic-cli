package cmd

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/quicdb/quic-cli/internal/api"
	"github.com/quicdb/quic-cli/internal/auth"
	"github.com/quicdb/quic-cli/internal/config"
	"github.com/spf13/cobra"
)

// TokenResponse represents the response from the token exchange
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
	TokenType    string `json:"token_type"`
	IDToken      string `json:"id_token"`
	RequestID    string `json:"request_id"`
	StatusCode   int    `json:"status_code"`
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login with QuicDB",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := config.Get()

		// Check for M2M authentication flags
		clientID, _ := cmd.Flags().GetString("client-id")
		clientSecret, _ := cmd.Flags().GetString("client-secret")

		if clientID != "" && clientSecret != "" {
			// M2M authentication flow
			if err := loginM2M(cfg, clientID, clientSecret); err != nil {
				fmt.Printf("M2M login failed: %v\n", err)
				return
			}
			fmt.Println("You're logged in!")
			return
		}

		if clientID != "" || clientSecret != "" {
			fmt.Println("Error: Both --client-id and --client-secret are required for M2M authentication")
			return
		}

		// Standard OAuth/PKCE flow
		port := getOpenPort()
		redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

		// Generate PKCE values
		codeVerifier, err := generateCodeVerifier()
		if err != nil {
			fmt.Printf("Error generating code verifier: %v\n", err)
			return
		}
		codeChallenge := generateCodeChallenge(codeVerifier)

		// Start local server to receive the callback
		server := &http.Server{
			Addr: fmt.Sprintf(":%d", port),
		}

		// Channel to receive the auth code
		codeChan := make(chan string)
		errorChan := make(chan error)

		// Set up the callback handler
		http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
			code := r.URL.Query().Get("code")
			if code == "" {
				errorChan <- fmt.Errorf("no code received in callback")
				return
			}

			// Send success response to browser
			w.Write([]byte("Authentication successful! You can close this window."))
			codeChan <- code
		})

		// Start the server in a goroutine
		go func() {
			if err := server.ListenAndServe(); err != http.ErrServerClosed {
				errorChan <- err
			}
		}()

		// Construct the auth URL with PKCE parameters
		params := url.Values{}
		params.Add("client_id", cfg.ClientID)
		params.Add("redirect_uri", redirectURI)
		params.Add("response_type", "code")
		params.Add("code_challenge", codeChallenge)
		params.Add("code_challenge_method", "S256")
		params.Add("scope", "offline_access")

		authURL := fmt.Sprintf("%s?%s", cfg.AuthorizeURL, params.Encode())

		fmt.Println("Opening browser for authentication...")

		// Open the browser with the auth URL
		err = openBrowser(authURL)
		if err != nil {
			fmt.Println("Please open the following URL in your browser:", authURL)
		}

		// Wait for either the code or an error
		var code string
		select {
		case code = <-codeChan:
			// Received authorization code
		case err := <-errorChan:
			fmt.Printf("Error: %v\n", err)
			return
		case <-time.After(5 * time.Minute):
			fmt.Println("Timeout waiting for authentication")
			return
		}

		// Exchange the code for a token
		token, err := exchangeCodeForToken(cfg.ClientID, cfg.ProjectID, code, codeVerifier, cfg.StytchURL)
		if err != nil {
			fmt.Printf("Error exchanging code for token: %v\n", err)
			return
		}

		// Save tokens securely to OS keychain/credential manager
		if err := auth.SaveToken(token.AccessToken, auth.AccessToken); err != nil {
			fmt.Printf("Warning: Failed to save access token: %v\n", err)
		}

		if token.RefreshToken != "" {
			if err := auth.SaveToken(token.RefreshToken, auth.RefreshToken); err != nil {
				fmt.Printf("Warning: Failed to save refresh token: %v\n", err)
			}
		}

		fmt.Println("You're logged in!")

		// Shutdown the server immediately
		server.Close()
	},
}

func init() {
	loginCmd.Flags().String("client-id", "", "M2M client ID for CI/CD authentication")
	loginCmd.Flags().String("client-secret", "", "M2M client secret for CI/CD authentication")
}

// generateCodeVerifier generates a random code verifier for PKCE
func generateCodeVerifier() (string, error) {
	// Generate 32 random bytes
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Base64URL encode the bytes
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// generateCodeChallenge generates a code challenge from a verifier
func generateCodeChallenge(verifier string) string {
	// Hash the verifier with SHA-256
	h := sha256.New()
	h.Write([]byte(verifier))
	// Base64URL encode the hash
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

// getOpenPort finds an available port
func getOpenPort() int {
	// Ask the OS for a free TCP port on all interfaces.
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	// Extract the chosen port.
	addr := ln.Addr().(*net.TCPAddr)
	port := addr.Port
	return port
}

// openBrowser opens a URL in the default browser
func openBrowser(url string) error {
	var cmd string
	var args []string

	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
		args = []string{url}
	case "windows":
		cmd = "rundll32"
		args = []string{"url.dll,FileProtocolHandler", url}
	default: // linux, freebsd, etc.
		cmd = "xdg-open"
		args = []string{url}
	}

	return exec.Command(cmd, args...).Start()
}

// loginM2M performs M2M authentication using client credentials
func loginM2M(cfg *config.Config, clientID, clientSecret string) error {
	// Exchange M2M credentials for tokens
	tokenResp, err := api.ExchangeM2MToken(clientID, clientSecret, cfg.StytchURL, cfg.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to exchange m2m token: %v", err)
	}

	// Save M2M credentials for automatic token refresh
	if err := auth.SaveM2MCredentials(clientID, clientSecret); err != nil {
		return fmt.Errorf("failed to save M2M credentials: %v", err)
	}

	// Save access token
	if err := auth.SaveToken(tokenResp.AccessToken, auth.AccessToken); err != nil {
		return fmt.Errorf("failed to save access token: %v", err)
	}

	// M2M tokens don't include refresh tokens, but check just in case
	if tokenResp.RefreshToken != "" {
		if err := auth.SaveToken(tokenResp.RefreshToken, auth.RefreshToken); err != nil {
			fmt.Printf("Warning: Failed to save refresh token: %v\n", err)
		}
	}

	return nil
}

// exchangeCodeForToken exchanges the authorization code for an access token
func exchangeCodeForToken(clientID, projectID, code, codeVerifier, stytchURL string) (*TokenResponse, error) {
	url := fmt.Sprintf("%s/v1/public/%s/oauth2/token", stytchURL, projectID)

	// Prepare the request body
	body := map[string]string{
		"client_id":     clientID,
		"code_verifier": codeVerifier,
		"code":          code,
		"grant_type":    "authorization_code",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request body: %v", err)
	}

	// Create the request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
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
	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("error parsing response: %v", err)
	}

	if tokenResp.StatusCode != 200 {
		return nil, fmt.Errorf("error from API: %s", string(respBody))
	}

	return &tokenResp, nil
}
