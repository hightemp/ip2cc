package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hightemp/ip2cc/internal/provider"
)

func TestLookupResultFormatText(t *testing.T) {
	result := &LookupResult{
		IP:          "8.8.8.8",
		CountryCode: "US",
		CountryName: "United States",
		Network:     "8.8.8.0/24",
		Provider: &provider.Result{
			Holders: []string{"GOOGLE LLC"},
		},
	}

	text := result.FormatText()

	// Check tab-separated format
	parts := strings.Split(text, "\t")
	if len(parts) != 5 {
		t.Errorf("Expected 5 tab-separated parts, got %d", len(parts))
	}

	if parts[0] != "8.8.8.8" {
		t.Errorf("IP = %s, expected 8.8.8.8", parts[0])
	}
	if parts[1] != "US" {
		t.Errorf("CountryCode = %s, expected US", parts[1])
	}
	if parts[2] != "United States" {
		t.Errorf("CountryName = %s, expected United States", parts[2])
	}
	if parts[3] != "8.8.8.0/24" {
		t.Errorf("Network = %s, expected 8.8.8.0/24", parts[3])
	}
	if parts[4] != "GOOGLE LLC" {
		t.Errorf("Provider = %s, expected GOOGLE LLC", parts[4])
	}
}

func TestLookupResultFormatTextError(t *testing.T) {
	result := &LookupResult{
		IP:    "invalid",
		Error: "invalid IP address",
	}

	text := result.FormatText()

	if !strings.Contains(text, "ERROR:") {
		t.Error("Error result should contain ERROR:")
	}
	if !strings.Contains(text, "invalid IP address") {
		t.Error("Error result should contain error message")
	}
}

func TestLookupResultFormatTextNoProvider(t *testing.T) {
	result := &LookupResult{
		IP:          "8.8.8.8",
		CountryCode: "US",
		CountryName: "United States",
		Network:     "8.8.8.0/24",
		Provider:    nil,
	}

	text := result.FormatText()

	if !strings.Contains(text, "unknown") {
		t.Error("Missing provider should show 'unknown'")
	}
}

func TestLookupResultFormatJSON(t *testing.T) {
	now := time.Now()
	result := &LookupResult{
		IP:           "8.8.8.8",
		CountryCode:  "US",
		CountryName:  "United States",
		Network:      "8.8.8.0/24",
		SnapshotTime: "2025-01-15",
		IndexBuiltAt: now,
		Provider: &provider.Result{
			Mode:    provider.ModeBGP,
			ASNs:    []int{15169},
			Holders: []string{"GOOGLE LLC"},
			Source:  "RIPEstat",
			Cached:  false,
		},
	}

	jsonStr, err := result.FormatJSON()
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Parse and verify JSON structure
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	if parsed["ip"] != "8.8.8.8" {
		t.Errorf("ip = %v, expected 8.8.8.8", parsed["ip"])
	}
	if parsed["country_code"] != "US" {
		t.Errorf("country_code = %v, expected US", parsed["country_code"])
	}
	if parsed["network"] != "8.8.8.0/24" {
		t.Errorf("network = %v, expected 8.8.8.0/24", parsed["network"])
	}
}

func TestBatchResultFormatText(t *testing.T) {
	batch := &BatchResult{
		Results: []*LookupResult{
			{
				IP:          "8.8.8.8",
				CountryCode: "US",
				CountryName: "United States",
				Network:     "8.8.8.0/24",
				Provider:    &provider.Result{Holders: []string{"GOOGLE LLC"}},
			},
			{
				IP:          "1.1.1.1",
				CountryCode: "AU",
				CountryName: "Australia",
				Network:     "1.1.1.0/24",
				Provider:    &provider.Result{Holders: []string{"CLOUDFLARE INC"}},
			},
		},
	}

	text := batch.FormatText()
	lines := strings.Split(text, "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "8.8.8.8") {
		t.Error("First line should contain first IP")
	}
	if !strings.Contains(lines[1], "1.1.1.1") {
		t.Error("Second line should contain second IP")
	}
}

func TestBatchResultFormatJSON(t *testing.T) {
	batch := &BatchResult{
		Results: []*LookupResult{
			{
				IP:          "8.8.8.8",
				CountryCode: "US",
				CountryName: "United States",
				Network:     "8.8.8.0/24",
			},
		},
	}

	jsonStr, err := batch.FormatJSON()
	if err != nil {
		t.Fatalf("FormatJSON failed: %v", err)
	}

	// Should be a JSON array
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		t.Fatalf("Invalid JSON array: %v", err)
	}

	if len(parsed) != 1 {
		t.Errorf("Expected 1 result, got %d", len(parsed))
	}
}

func TestFormatError(t *testing.T) {
	err := &customError{msg: "test error"}
	text := FormatError("8.8.8.8", err)

	if !strings.Contains(text, "8.8.8.8") {
		t.Error("Error line should contain IP")
	}
	if !strings.Contains(text, "ERROR:") {
		t.Error("Error line should contain ERROR:")
	}
	if !strings.Contains(text, "test error") {
		t.Error("Error line should contain error message")
	}
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
