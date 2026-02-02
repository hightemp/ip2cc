package provider

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseMode(t *testing.T) {
	tests := []struct {
		input    string
		expected Mode
		hasError bool
	}{
		{"bgp", ModeBGP, false},
		{"", ModeBGP, false},
		{"whois", ModeWhois, false},
		{"off", ModeOff, false},
		{"invalid", "", true},
		{"BGP", "", true}, // Case sensitive
	}

	for _, tc := range tests {
		mode, err := ParseMode(tc.input)
		if tc.hasError {
			if err == nil {
				t.Errorf("ParseMode(%q) expected error", tc.input)
			}
		} else {
			if err != nil {
				t.Errorf("ParseMode(%q) unexpected error: %v", tc.input, err)
			}
			if mode != tc.expected {
				t.Errorf("ParseMode(%q) = %q, expected %q", tc.input, mode, tc.expected)
			}
		}
	}
}

func TestCacheSetAndGet(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := NewCache(cachePath, 7)

	// Set value
	cache.Set(15169, "GOOGLE LLC")

	// Get value
	holder, ok := cache.Get(15169)
	if !ok {
		t.Error("Expected to find cached value")
	}
	if holder != "GOOGLE LLC" {
		t.Errorf("Holder = %q, expected %q", holder, "GOOGLE LLC")
	}

	// Get non-existent
	_, ok = cache.Get(99999)
	if ok {
		t.Error("Expected not to find non-existent ASN")
	}
}

func TestCacheSaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create and populate cache
	cache := NewCache(cachePath, 7)
	cache.Set(15169, "GOOGLE LLC")
	cache.Set(13335, "CLOUDFLARE INC")

	// Save
	if err := cache.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Create new cache and load
	cache2 := NewCache(cachePath, 7)
	if err := cache2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Verify values
	holder, ok := cache2.Get(15169)
	if !ok || holder != "GOOGLE LLC" {
		t.Errorf("Loaded cache missing value for 15169")
	}

	holder, ok = cache2.Get(13335)
	if !ok || holder != "CLOUDFLARE INC" {
		t.Errorf("Loaded cache missing value for 13335")
	}
}

func TestCacheExpiration(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")

	// Create cache with 0 day TTL (immediate expiration)
	cache := NewCache(cachePath, 0)
	cache.Set(15169, "GOOGLE LLC")

	// Wait a moment
	time.Sleep(10 * time.Millisecond)

	// Should be expired
	_, ok := cache.Get(15169)
	if ok {
		t.Error("Expected cache entry to be expired")
	}
}

func TestCacheCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := NewCache(cachePath, 0) // 0 days = immediate expiration

	cache.Set(15169, "GOOGLE LLC")
	cache.Set(13335, "CLOUDFLARE INC")

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Cleanup should remove expired entries
	removed := cache.Cleanup()
	if removed != 2 {
		t.Errorf("Cleanup removed %d entries, expected 2", removed)
	}

	if cache.Size() != 0 {
		t.Errorf("Cache size = %d, expected 0", cache.Size())
	}
}

func TestCacheClear(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cachePath := filepath.Join(tmpDir, "cache.json")
	cache := NewCache(cachePath, 7)

	cache.Set(15169, "GOOGLE LLC")
	cache.Set(13335, "CLOUDFLARE INC")

	cache.Clear()

	if cache.Size() != 0 {
		t.Errorf("Cache size = %d, expected 0 after clear", cache.Size())
	}
}

func TestCacheLoadNonexistent(t *testing.T) {
	cache := NewCache("/nonexistent/path/cache.json", 7)
	err := cache.Load()
	// Should not error, just return empty cache
	if err != nil {
		t.Errorf("Load nonexistent file should not error: %v", err)
	}
	if cache.Size() != 0 {
		t.Errorf("Cache size = %d, expected 0", cache.Size())
	}
}

func TestResultGetHolderString(t *testing.T) {
	tests := []struct {
		result   *Result
		expected string
	}{
		{
			&Result{Holders: []string{"GOOGLE LLC", "Other"}},
			"GOOGLE LLC",
		},
		{
			&Result{Holders: []string{}},
			"unknown",
		},
		{
			&Result{Holders: nil, Error: "some error"},
			"unknown",
		},
	}

	for i, tc := range tests {
		got := tc.result.GetHolderString()
		if got != tc.expected {
			t.Errorf("Test %d: GetHolderString() = %q, expected %q", i, got, tc.expected)
		}
	}
}
