package config

// These values are injected at build time via -ldflags
var (
	ClientID     = ""
	ProjectID    = ""
	AuthorizeURL = "http://localhost:5173/oauth/authorize"
	StytchURL    = "https://test.stytch.com"
	APIURL       = "http://localhost:8080"
)

// Config holds all configuration values
type Config struct {
	ClientID     string
	ProjectID    string
	AuthorizeURL string
	StytchURL    string
	APIURL       string
}

// Get returns the current configuration
func Get() *Config {
	return &Config{
		ClientID:     ClientID,
		ProjectID:    ProjectID,
		AuthorizeURL: AuthorizeURL,
		StytchURL:    StytchURL,
		APIURL:       APIURL,
	}
}
