package countries

import (
	"testing"
)

func TestGetName(t *testing.T) {
	tests := []struct {
		code     string
		expected string
	}{
		{"US", "United States"},
		{"us", "United States"},
		{"GB", "United Kingdom"},
		{"DE", "Germany"},
		{"JP", "Japan"},
		{"CN", "China"},
		{"AU", "Australia"},
		{"XX", ""}, // Invalid code
		{"", ""},   // Empty code
	}

	for _, tc := range tests {
		result := GetName(tc.code)
		if result != tc.expected {
			t.Errorf("GetName(%q) = %q, expected %q", tc.code, result, tc.expected)
		}
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{"US", true},
		{"us", true},
		{"GB", true},
		{"XX", false},
		{"", false},
		{"USA", false},
		{"U", false},
	}

	for _, tc := range tests {
		result := IsValid(tc.code)
		if result != tc.expected {
			t.Errorf("IsValid(%q) = %v, expected %v", tc.code, result, tc.expected)
		}
	}
}

func TestAllCodes(t *testing.T) {
	codes := AllCodes()

	if len(codes) < 200 {
		t.Errorf("Expected at least 200 country codes, got %d", len(codes))
	}

	// Check some expected codes
	found := make(map[string]bool)
	for _, c := range codes {
		found[c] = true
	}

	expectedCodes := []string{"US", "GB", "DE", "FR", "CN", "JP", "AU"}
	for _, c := range expectedCodes {
		if !found[c] {
			t.Errorf("Expected code %s not found in AllCodes()", c)
		}
	}
}

func TestAllCodesLower(t *testing.T) {
	codes := AllCodesLower()

	for _, c := range codes {
		if c != c {
			t.Errorf("Code %s is not lowercase", c)
		}
		if len(c) != 2 {
			t.Errorf("Code %s has wrong length", c)
		}
	}

	// Check it matches AllCodes count
	if len(codes) != len(AllCodes()) {
		t.Errorf("AllCodesLower() count %d != AllCodes() count %d", len(codes), len(AllCodes()))
	}
}

func TestCount(t *testing.T) {
	count := Count()
	if count < 200 {
		t.Errorf("Expected at least 200 countries, got %d", count)
	}
}

func TestLoadFromFile(t *testing.T) {
	content := `# Comment line
US
GB
de

FR
invalid
XX
`
	codes, err := LoadFromFile(content)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	expected := []string{"us", "gb", "de", "fr", "xx"}
	if len(codes) != len(expected) {
		t.Errorf("Expected %d codes, got %d", len(expected), len(codes))
	}

	for i, c := range codes {
		if c != expected[i] {
			t.Errorf("Code %d: got %s, expected %s", i, c, expected[i])
		}
	}
}

func TestLoadFromFileEmpty(t *testing.T) {
	codes, err := LoadFromFile("")
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if len(codes) != 0 {
		t.Errorf("Expected 0 codes, got %d", len(codes))
	}
}
