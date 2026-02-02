package ripestat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientGet(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		// Check sourceapp parameter
		if r.URL.Query().Get("sourceapp") != "ip2cc" {
			t.Error("Missing or incorrect sourceapp parameter")
		}

		// Return mock response
		resp := Response{
			Status:     "ok",
			StatusCode: 200,
			Data:       json.RawMessage(`{"test": "data"}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	ctx := context.Background()
	resp, err := client.Get(ctx, "test-endpoint", nil)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Status = %s, expected ok", resp.Status)
	}
}

func TestClientGetRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			// Fail first 2 attempts
			http.Error(w, "Server Error", http.StatusInternalServerError)
			return
		}
		// Succeed on 3rd attempt
		resp := Response{
			Status:     "ok",
			StatusCode: 200,
			Data:       json.RawMessage(`{}`),
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	ctx := context.Background()
	_, err := client.Get(ctx, "test", nil)
	if err != nil {
		t.Fatalf("Get failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestClientGetTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	client := NewClientWithTimeout(100 * time.Millisecond)
	client.baseURL = server.URL

	ctx := context.Background()
	_, err := client.Get(ctx, "test", nil)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

func TestClientGetContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "test", nil)
	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestClientGetAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := Response{
			Status:     "error",
			StatusCode: 400,
			Messages:   [][]string{{"error", "Test error message"}},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient()
	client.baseURL = server.URL

	ctx := context.Background()
	_, err := client.Get(ctx, "test", nil)
	if err == nil {
		t.Error("Expected API error")
	}
}

func TestCalculateBackoff(t *testing.T) {
	client := NewClient()

	// Test that backoff increases with attempts
	b1 := client.calculateBackoff(1)
	b2 := client.calculateBackoff(2)
	b3 := client.calculateBackoff(3)

	// Backoff should generally increase (accounting for jitter)
	if b2 < b1/2 {
		t.Errorf("Backoff should increase: b1=%v, b2=%v", b1, b2)
	}
	if b3 < b2/2 {
		t.Errorf("Backoff should increase: b2=%v, b3=%v", b2, b3)
	}

	// Should not exceed MaxBackoff
	b10 := client.calculateBackoff(10)
	if b10 > MaxBackoff+MaxBackoff/4 {
		t.Errorf("Backoff exceeded max: %v > %v", b10, MaxBackoff)
	}
}
