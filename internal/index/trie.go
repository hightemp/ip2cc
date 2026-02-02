// Package index provides Patricia trie implementation for IP prefix lookup.
package index

import (
	"fmt"
	"net/netip"
	"strings"
)

// PrefixData holds data associated with a prefix.
type PrefixData struct {
	CountryCode string
	PrefixStr   string // Original CIDR string
}

// TrieNode represents a node in the Patricia trie.
type TrieNode struct {
	// Prefix bits for this node (path compression)
	Prefix []byte
	// Number of significant bits in Prefix
	PrefixLen int
	// Data if this node represents a complete prefix
	Data *PrefixData
	// Children: 0 for bit=0, 1 for bit=1
	Children [2]*TrieNode
}

// Trie is a Patricia trie for IP prefix lookup.
type Trie struct {
	Root   *TrieNode
	IsIPv6 bool
	Count  int
}

// NewTrie creates a new empty trie.
func NewTrie(isIPv6 bool) *Trie {
	return &Trie{
		Root:   &TrieNode{},
		IsIPv6: isIPv6,
	}
}

// Insert adds a prefix with associated data to the trie.
func (t *Trie) Insert(prefix netip.Prefix, data PrefixData) error {
	if prefix.Addr().Is6() != t.IsIPv6 {
		return fmt.Errorf("IP version mismatch")
	}

	bits := prefixToBits(prefix)
	prefixLen := prefix.Bits()

	t.insertRecursive(t.Root, bits, 0, prefixLen, &data)
	t.Count++
	return nil
}

// insertRecursive inserts data at the given path in the trie
func (t *Trie) insertRecursive(node *TrieNode, bits []byte, pos int, prefixLen int, data *PrefixData) {
	for {
		if pos == prefixLen {
			// We've reached the end of the prefix - store data here
			node.Data = data
			return
		}

		// Get the next bit
		bit := getBit(bits, pos)
		child := node.Children[bit]

		if child == nil {
			// No child, create new node with remaining bits
			newNode := &TrieNode{
				Prefix:    extractBits(bits, pos, prefixLen-pos),
				PrefixLen: prefixLen - pos,
				Data:      data,
			}
			node.Children[bit] = newNode
			return
		}

		// There's a child, check for common prefix
		childBits := child.Prefix
		childLen := child.PrefixLen
		remainingLen := prefixLen - pos

		// Find common prefix length
		commonLen := 0
		maxCheck := min(remainingLen, childLen)
		for i := 0; i < maxCheck; i++ {
			if getBit(bits, pos+i) != getBitFromSlice(childBits, i) {
				break
			}
			commonLen++
		}

		if commonLen == childLen {
			// Child prefix is fully matched
			if commonLen == remainingLen {
				// Exact match - store data at child
				child.Data = data
				return
			}
			// Continue down this child
			pos += childLen
			node = child
			continue
		}

		// Need to split the child node
		// Create new intermediate node
		newParent := &TrieNode{
			Prefix:    extractBits(bits, pos, commonLen),
			PrefixLen: commonLen,
		}

		// Update old child to have only the remaining bits
		oldChildBit := getBitFromSlice(childBits, commonLen)
		child.Prefix = extractBitsFromSlice(childBits, commonLen, childLen-commonLen)
		child.PrefixLen = childLen - commonLen
		newParent.Children[oldChildBit] = child

		// Replace child with new parent
		node.Children[bit] = newParent

		if commonLen == remainingLen {
			// Our prefix ends at the split point
			newParent.Data = data
			return
		}

		// Create new leaf for our prefix
		newLeafBit := getBit(bits, pos+commonLen)
		newLeaf := &TrieNode{
			Prefix:    extractBits(bits, pos+commonLen, remainingLen-commonLen),
			PrefixLen: remainingLen - commonLen,
			Data:      data,
		}
		newParent.Children[newLeafBit] = newLeaf
		return
	}
}

// InsertCIDR parses a CIDR string and inserts it into the trie.
func (t *Trie) InsertCIDR(cidr string, countryCode string) error {
	prefix, err := netip.ParsePrefix(cidr)
	if err != nil {
		return fmt.Errorf("invalid CIDR %q: %w", cidr, err)
	}

	data := PrefixData{
		CountryCode: strings.ToUpper(countryCode),
		PrefixStr:   prefix.Masked().String(),
	}

	return t.Insert(prefix.Masked(), data)
}

// Lookup finds the longest matching prefix for an IP address.
func (t *Trie) Lookup(ip netip.Addr) *PrefixData {
	if ip.Is6() != t.IsIPv6 {
		return nil
	}

	bits := addrToBits(ip)
	maxBits := 32
	if t.IsIPv6 {
		maxBits = 128
	}

	return t.lookupRecursive(t.Root, bits, 0, maxBits)
}

func (t *Trie) lookupRecursive(node *TrieNode, bits []byte, pos int, maxBits int) *PrefixData {
	var lastMatch *PrefixData

	for node != nil {
		// Check if this node has data (it's a valid prefix end)
		if node.Data != nil {
			lastMatch = node.Data
		}

		if pos >= maxBits {
			break
		}

		// Get next bit to decide which child
		bit := getBit(bits, pos)
		child := node.Children[bit]

		if child == nil {
			break
		}

		// Check if the child's prefix matches our bits
		childLen := child.PrefixLen
		if pos+childLen > maxBits {
			childLen = maxBits - pos
		}

		// Verify all bits in child's prefix match
		matches := true
		for i := 0; i < childLen && i < child.PrefixLen; i++ {
			if getBit(bits, pos+i) != getBitFromSlice(child.Prefix, i) {
				matches = false
				break
			}
		}

		if !matches {
			break
		}

		pos += child.PrefixLen
		node = child
	}

	return lastMatch
}

// LookupString parses an IP string and looks it up.
func (t *Trie) LookupString(ipStr string) (*PrefixData, error) {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, fmt.Errorf("invalid IP %q: %w", ipStr, err)
	}
	return t.Lookup(ip), nil
}

// Helper functions

func prefixToBits(prefix netip.Prefix) []byte {
	addr := prefix.Addr()
	if addr.Is4() {
		a4 := addr.As4()
		result := make([]byte, 4)
		copy(result, a4[:])
		return result
	}
	a16 := addr.As16()
	result := make([]byte, 16)
	copy(result, a16[:])
	return result
}

func addrToBits(addr netip.Addr) []byte {
	if addr.Is4() {
		a4 := addr.As4()
		result := make([]byte, 4)
		copy(result, a4[:])
		return result
	}
	a16 := addr.As16()
	result := make([]byte, 16)
	copy(result, a16[:])
	return result
}

func getBit(data []byte, pos int) int {
	byteIdx := pos / 8
	bitIdx := 7 - (pos % 8)
	if byteIdx >= len(data) {
		return 0
	}
	return int((data[byteIdx] >> bitIdx) & 1)
}

func getBitFromSlice(data []byte, pos int) int {
	return getBit(data, pos)
}

func extractBits(data []byte, start, length int) []byte {
	if length <= 0 {
		return nil
	}
	result := make([]byte, (length+7)/8)
	for i := 0; i < length; i++ {
		bit := getBit(data, start+i)
		if bit == 1 {
			byteIdx := i / 8
			bitIdx := 7 - (i % 8)
			result[byteIdx] |= 1 << bitIdx
		}
	}
	return result
}

func extractBitsFromSlice(data []byte, start, length int) []byte {
	return extractBits(data, start, length)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
