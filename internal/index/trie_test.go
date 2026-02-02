package index

import (
	"net/netip"
	"testing"
)

func TestTrieInsertAndLookupIPv4(t *testing.T) {
	trie := NewTrie(false)

	// Insert some prefixes
	testCases := []struct {
		cidr        string
		countryCode string
	}{
		{"8.8.8.0/24", "US"},
		{"8.8.0.0/16", "US"},
		{"1.0.0.0/8", "AU"},
		{"1.2.3.0/24", "CN"},
		{"192.168.0.0/16", "ZZ"},
		{"192.168.1.0/24", "ZZ"},
	}

	for _, tc := range testCases {
		if err := trie.InsertCIDR(tc.cidr, tc.countryCode); err != nil {
			t.Fatalf("InsertCIDR(%s, %s) failed: %v", tc.cidr, tc.countryCode, err)
		}
	}

	// Test lookups
	lookupTests := []struct {
		ip             string
		expectedCC     string
		expectedPrefix string
	}{
		{"8.8.8.8", "US", "8.8.8.0/24"},           // Most specific match
		{"8.8.4.4", "US", "8.8.0.0/16"},           // Less specific match
		{"1.2.3.4", "CN", "1.2.3.0/24"},           // Most specific match
		{"1.1.1.1", "AU", "1.0.0.0/8"},            // Less specific match
		{"192.168.1.100", "ZZ", "192.168.1.0/24"}, // Most specific match
		{"192.168.2.100", "ZZ", "192.168.0.0/16"}, // Less specific match
	}

	for _, lt := range lookupTests {
		ip := netip.MustParseAddr(lt.ip)
		result := trie.Lookup(ip)
		if result == nil {
			t.Errorf("Lookup(%s) returned nil, expected %s", lt.ip, lt.expectedCC)
			continue
		}
		if result.CountryCode != lt.expectedCC {
			t.Errorf("Lookup(%s) CountryCode = %s, expected %s", lt.ip, result.CountryCode, lt.expectedCC)
		}
		if result.PrefixStr != lt.expectedPrefix {
			t.Errorf("Lookup(%s) PrefixStr = %s, expected %s", lt.ip, result.PrefixStr, lt.expectedPrefix)
		}
	}
}

func TestTrieInsertAndLookupIPv6(t *testing.T) {
	trie := NewTrie(true)

	// Insert some prefixes
	testCases := []struct {
		cidr        string
		countryCode string
	}{
		{"2001:4860:4860::/48", "US"},
		{"2001:4860::/32", "US"},
		{"2a00:1450::/32", "IE"},
		{"2a00:1450:4000::/36", "IE"},
	}

	for _, tc := range testCases {
		if err := trie.InsertCIDR(tc.cidr, tc.countryCode); err != nil {
			t.Fatalf("InsertCIDR(%s, %s) failed: %v", tc.cidr, tc.countryCode, err)
		}
	}

	// Test lookups
	lookupTests := []struct {
		ip             string
		expectedCC     string
		expectedPrefix string
	}{
		{"2001:4860:4860::8888", "US", "2001:4860:4860::/48"},
		{"2001:4860:1234::1", "US", "2001:4860::/32"},
		{"2a00:1450:4001::1", "IE", "2a00:1450:4000::/36"},
	}

	for _, lt := range lookupTests {
		ip := netip.MustParseAddr(lt.ip)
		result := trie.Lookup(ip)
		if result == nil {
			t.Errorf("Lookup(%s) returned nil, expected %s", lt.ip, lt.expectedCC)
			continue
		}
		if result.CountryCode != lt.expectedCC {
			t.Errorf("Lookup(%s) CountryCode = %s, expected %s", lt.ip, result.CountryCode, lt.expectedCC)
		}
		if result.PrefixStr != lt.expectedPrefix {
			t.Errorf("Lookup(%s) PrefixStr = %s, expected %s", lt.ip, result.PrefixStr, lt.expectedPrefix)
		}
	}
}

func TestTrieLookupNotFound(t *testing.T) {
	trie := NewTrie(false)

	if err := trie.InsertCIDR("10.0.0.0/8", "ZZ"); err != nil {
		t.Fatalf("InsertCIDR failed: %v", err)
	}

	// IP not in any prefix
	ip := netip.MustParseAddr("192.168.1.1")
	result := trie.Lookup(ip)
	if result != nil {
		t.Errorf("Lookup(192.168.1.1) expected nil, got %+v", result)
	}
}

func TestTrieIPVersionMismatch(t *testing.T) {
	v4Trie := NewTrie(false)
	v6Trie := NewTrie(true)

	// Try to insert IPv6 into IPv4 trie
	err := v4Trie.InsertCIDR("2001:db8::/32", "US")
	if err == nil {
		t.Error("Expected error when inserting IPv6 into IPv4 trie")
	}

	// Try to insert IPv4 into IPv6 trie
	err = v6Trie.InsertCIDR("8.8.8.0/24", "US")
	if err == nil {
		t.Error("Expected error when inserting IPv4 into IPv6 trie")
	}

	// Lookup IPv6 in IPv4 trie
	ip6 := netip.MustParseAddr("2001:db8::1")
	result := v4Trie.Lookup(ip6)
	if result != nil {
		t.Error("Expected nil when looking up IPv6 in IPv4 trie")
	}

	// Lookup IPv4 in IPv6 trie
	ip4 := netip.MustParseAddr("8.8.8.8")
	result = v6Trie.Lookup(ip4)
	if result != nil {
		t.Error("Expected nil when looking up IPv4 in IPv6 trie")
	}
}

func TestTrieLookupString(t *testing.T) {
	trie := NewTrie(false)

	if err := trie.InsertCIDR("8.8.8.0/24", "US"); err != nil {
		t.Fatalf("InsertCIDR failed: %v", err)
	}

	// Valid IP
	result, err := trie.LookupString("8.8.8.8")
	if err != nil {
		t.Errorf("LookupString(8.8.8.8) error: %v", err)
	}
	if result == nil || result.CountryCode != "US" {
		t.Errorf("LookupString(8.8.8.8) unexpected result: %+v", result)
	}

	// Invalid IP
	_, err = trie.LookupString("not-an-ip")
	if err == nil {
		t.Error("Expected error for invalid IP")
	}
}

func TestTrieInvalidCIDR(t *testing.T) {
	trie := NewTrie(false)

	invalidCIDRs := []string{
		"not-a-cidr",
		"8.8.8.8",          // No prefix length
		"8.8.8.8/33",       // Invalid prefix length
		"256.256.256.0/24", // Invalid IP
	}

	for _, cidr := range invalidCIDRs {
		err := trie.InsertCIDR(cidr, "US")
		if err == nil {
			t.Errorf("InsertCIDR(%s) should have failed", cidr)
		}
	}
}

func BenchmarkTrieInsertIPv4(b *testing.B) {
	trie := NewTrie(false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a := byte(i % 256)
		c := byte((i / 256) % 256)
		cidr := netip.PrefixFrom(netip.AddrFrom4([4]byte{a, c, 0, 0}), 16)
		trie.Insert(cidr, PrefixData{CountryCode: "US", PrefixStr: cidr.String()})
	}
}

func BenchmarkTrieLookupIPv4(b *testing.B) {
	trie := NewTrie(false)

	// Pre-populate with some prefixes
	for i := 0; i < 1000; i++ {
		a := byte(i % 256)
		c := byte((i / 256) % 256)
		trie.InsertCIDR(netip.PrefixFrom(netip.AddrFrom4([4]byte{a, c, 0, 0}), 16).String(), "US")
	}

	ip := netip.MustParseAddr("100.50.25.1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.Lookup(ip)
	}
}
