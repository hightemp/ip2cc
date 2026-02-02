// Package ripestat provides HTTP client for RIPEstat Data API.
package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"time"

	"github.com/hightemp/ip2cc/internal/config"
)

const (
	// BaseURL is the RIPEstat Data API base URL.
	BaseURL = "https://stat.ripe.net/data"

	// DefaultTimeout for HTTP requests.
	DefaultTimeout = 30 * time.Second

	// MaxRetries for failed requests.
	MaxRetries = 3

	// BaseBackoff for exponential backoff.
	BaseBackoff = 1 * time.Second

	// MaxBackoff for exponential backoff.
	MaxBackoff = 30 * time.Second
)

// Client is an HTTP client for RIPEstat API.
type Client struct {
	httpClient *http.Client
	sourceApp  string
	baseURL    string
}

// NewClient creates a new RIPEstat client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: DefaultTimeout,
		},
		sourceApp: config.RIPEstatSourceApp,
		baseURL:   BaseURL,
	}
}

// NewClientWithTimeout creates a new RIPEstat client with custom timeout.
func NewClientWithTimeout(timeout time.Duration) *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: timeout,
		},
		sourceApp: config.RIPEstatSourceApp,
		baseURL:   BaseURL,
	}
}

// Response is the generic RIPEstat API response wrapper.
type Response struct {
	Status         string          `json:"status"`
	StatusCode     int             `json:"status_code"`
	Data           json.RawMessage `json:"data"`
	Messages       [][]string      `json:"messages"`
	SeeAlso        []interface{}   `json:"see_also"`
	Version        string          `json:"version"`
	DataCallName   string          `json:"data_call_name"`
	DataCallStatus string          `json:"data_call_status"`
	Cached         bool            `json:"cached"`
	QueryID        string          `json:"query_id"`
	ProcessTime    int             `json:"process_time"`
	ServerID       string          `json:"server_id"`
	BuildVersion   string          `json:"build_version"`
	Time           string          `json:"time"`
}

// Get performs a GET request to the specified endpoint with retries.
func (c *Client) Get(ctx context.Context, endpoint string, params url.Values) (*Response, error) {
	if params == nil {
		params = url.Values{}
	}
	params.Set("sourceapp", c.sourceApp)

	fullURL := fmt.Sprintf("%s/%s/data.json?%s", c.baseURL, endpoint, params.Encode())

	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			backoff := c.calculateBackoff(attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		resp, err := c.doRequest(ctx, fullURL)
		if err == nil {
			return resp, nil
		}
		lastErr = err

		// Don't retry on context cancellation
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}

	return nil, fmt.Errorf("after %d retries: %w", MaxRetries, lastErr)
}

func (c *Client) doRequest(ctx context.Context, url string) (*Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ip2cc/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	var result Response
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.Status != "ok" {
		return nil, fmt.Errorf("API error: status=%s, messages=%v", result.Status, result.Messages)
	}

	return &result, nil
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	backoff := BaseBackoff * time.Duration(1<<uint(attempt-1))
	if backoff > MaxBackoff {
		backoff = MaxBackoff
	}
	// Add jitter (0-25% of backoff)
	jitter := time.Duration(rand.Int63n(int64(backoff / 4)))
	return backoff + jitter
}
