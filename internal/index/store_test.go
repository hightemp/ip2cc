package index

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadIndex(t *testing.T) {
	// Create temporary directory
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	v4Path := filepath.Join(tmpDir, "index_v4.bin")
	v6Path := filepath.Join(tmpDir, "index_v6.bin")

	// Create and populate tries
	v4Trie := NewTrie(false)
	v6Trie := NewTrie(true)

	v4Prefixes := []struct {
		cidr string
		cc   string
	}{
		{"8.8.8.0/24", "US"},
		{"8.8.0.0/16", "US"},
		{"1.0.0.0/8", "AU"},
		{"192.168.0.0/16", "ZZ"},
	}

	v6Prefixes := []struct {
		cidr string
		cc   string
	}{
		{"2001:4860::/32", "US"},
		{"2a00:1450::/32", "IE"},
	}

	for _, p := range v4Prefixes {
		if err := v4Trie.InsertCIDR(p.cidr, p.cc); err != nil {
			t.Fatalf("InsertCIDR failed: %v", err)
		}
	}

	for _, p := range v6Prefixes {
		if err := v6Trie.InsertCIDR(p.cidr, p.cc); err != nil {
			t.Fatalf("InsertCIDR failed: %v", err)
		}
	}

	// Save indices
	if err := SaveIndex(v4Path, v6Path, v4Trie, v6Trie); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Verify files exist
	if _, err := os.Stat(v4Path); os.IsNotExist(err) {
		t.Error("IPv4 index file not created")
	}
	if _, err := os.Stat(v6Path); os.IsNotExist(err) {
		t.Error("IPv6 index file not created")
	}

	// Load indices
	loadedV4, loadedV6, err := LoadIndex(v4Path, v6Path)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Verify loaded tries work correctly
	// Test IPv4 lookups
	result, err := loadedV4.LookupString("8.8.8.8")
	if err != nil {
		t.Errorf("Lookup error: %v", err)
	}
	if result == nil || result.CountryCode != "US" {
		t.Errorf("Loaded IPv4 trie lookup failed, got: %+v", result)
	}

	// Test IPv6 lookups
	result, err = loadedV6.LookupString("2001:4860::1")
	if err != nil {
		t.Errorf("Lookup error: %v", err)
	}
	if result == nil || result.CountryCode != "US" {
		t.Errorf("Loaded IPv6 trie lookup failed, got: %+v", result)
	}

	// Test prefix string is preserved
	result, err = loadedV4.LookupString("8.8.8.8")
	if err != nil {
		t.Errorf("Lookup error: %v", err)
	}
	if result.PrefixStr != "8.8.8.0/24" {
		t.Errorf("PrefixStr not preserved, got: %s", result.PrefixStr)
	}
}

func TestLoadIndexInvalidFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create invalid index file
	invalidPath := filepath.Join(tmpDir, "invalid.bin")
	os.WriteFile(invalidPath, []byte("not a valid index"), 0644)

	_, err = loadTrie(invalidPath, false)
	if err == nil {
		t.Error("Expected error loading invalid file")
	}
}

func TestLoadIndexNonexistent(t *testing.T) {
	_, err := loadTrie("/nonexistent/path/index.bin", false)
	if err == nil {
		t.Error("Expected error loading nonexistent file")
	}
}

func TestSaveAndLoadEmptyTrie(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	v4Path := filepath.Join(tmpDir, "empty_v4.bin")
	v6Path := filepath.Join(tmpDir, "empty_v6.bin")

	// Save empty tries
	v4Trie := NewTrie(false)
	v6Trie := NewTrie(true)

	if err := SaveIndex(v4Path, v6Path, v4Trie, v6Trie); err != nil {
		t.Fatalf("SaveIndex failed: %v", err)
	}

	// Load and verify
	loadedV4, loadedV6, err := LoadIndex(v4Path, v6Path)
	if err != nil {
		t.Fatalf("LoadIndex failed: %v", err)
	}

	// Lookup should return nil
	result, _ := loadedV4.LookupString("8.8.8.8")
	if result != nil {
		t.Error("Expected nil result from empty trie")
	}

	result, _ = loadedV6.LookupString("2001:db8::1")
	if result != nil {
		t.Error("Expected nil result from empty trie")
	}
}
