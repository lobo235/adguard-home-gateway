package adguard

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Rewrite represents a single DNS rewrite entry in AdGuard Home.
type Rewrite struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

// Client wraps the AdGuard Home REST API with HTTP Basic Auth.
type Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
}

// NewClient creates a new AdGuard Home API client.
// username and password may be empty if AdGuard Home has no authentication configured.
// Set tlsSkipVerify to true to skip TLS certificate verification (useful for self-signed certs).
func NewClient(baseURL, username, password string, tlsSkipVerify bool) *Client {
	transport := http.DefaultTransport
	if tlsSkipVerify {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
		}
	}
	return &Client{
		baseURL:  baseURL,
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout:   10 * time.Second,
			Transport: transport,
		},
	}
}

func (c *Client) newRequest(method, path string, body any) (*http.Request, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return req, nil
}

// Ping verifies connectivity to AdGuard Home by calling /control/status.
func (c *Client) Ping() error {
	req, err := c.newRequest(http.MethodGet, "/control/status", nil)
	if err != nil {
		return fmt.Errorf("creating ping request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("adguard ping failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adguard ping returned status %d", resp.StatusCode)
	}
	return nil
}

// ListRewrites returns all DNS rewrite entries from AdGuard Home.
func (c *Client) ListRewrites() ([]Rewrite, error) {
	req, err := c.newRequest(http.MethodGet, "/control/rewrite/list", nil)
	if err != nil {
		return nil, fmt.Errorf("creating list request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("listing rewrites: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("adguard returned status %d", resp.StatusCode)
	}
	var rewrites []Rewrite
	if err := json.NewDecoder(resp.Body).Decode(&rewrites); err != nil {
		return nil, fmt.Errorf("decoding rewrites: %w", err)
	}
	return rewrites, nil
}

// AddRewrite adds a new DNS rewrite entry to AdGuard Home.
func (c *Client) AddRewrite(domain, answer string) error {
	req, err := c.newRequest(http.MethodPost, "/control/rewrite/add", Rewrite{Domain: domain, Answer: answer})
	if err != nil {
		return fmt.Errorf("creating add request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("adding rewrite: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adguard returned status %d", resp.StatusCode)
	}
	return nil
}

// DeleteRewrite removes a DNS rewrite entry from AdGuard Home.
// Both domain and answer are required since AdGuard identifies entries by the pair.
func (c *Client) DeleteRewrite(domain, answer string) error {
	req, err := c.newRequest(http.MethodPost, "/control/rewrite/delete", Rewrite{Domain: domain, Answer: answer})
	if err != nil {
		return fmt.Errorf("creating delete request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("deleting rewrite: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("adguard returned status %d", resp.StatusCode)
	}
	return nil
}
