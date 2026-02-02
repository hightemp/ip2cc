// Package index provides binary serialization for the trie.
package index

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/hightemp/ip2cc/internal/config"
)

const (
	// Magic bytes for index file
	Magic = "IP2CCIDX"
	// Header size in bytes
	HeaderSize = 32
)

// Flags for index file
const (
	FlagHasIPv4 uint32 = 1 << iota
	FlagHasIPv6
)

// Header represents the index file header.
type Header struct {
	Magic      [8]byte
	Version    uint32
	Flags      uint32
	IPv4Offset uint64
	IPv6Offset uint64
}

// SerializedNode represents a node in serialized form.
type SerializedNode struct {
	PrefixLen    uint8
	Prefix       []byte
	HasData      bool
	CountryCode  [2]byte
	PrefixStrLen uint16
	PrefixStr    string
	LeftOffset   int32
	RightOffset  int32
}

// SaveIndex saves both IPv4 and IPv6 tries to files.
func SaveIndex(v4Path, v6Path string, v4Trie, v6Trie *Trie) error {
	if err := saveTrie(v4Path, v4Trie, false); err != nil {
		return fmt.Errorf("save IPv4 index: %w", err)
	}
	if err := saveTrie(v6Path, v6Trie, true); err != nil {
		return fmt.Errorf("save IPv6 index: %w", err)
	}
	return nil
}

// LoadIndex loads both IPv4 and IPv6 tries from files.
func LoadIndex(v4Path, v6Path string) (*Trie, *Trie, error) {
	v4Trie, err := loadTrie(v4Path, false)
	if err != nil {
		return nil, nil, fmt.Errorf("load IPv4 index: %w", err)
	}
	v6Trie, err := loadTrie(v6Path, true)
	if err != nil {
		return nil, nil, fmt.Errorf("load IPv6 index: %w", err)
	}
	return v4Trie, v6Trie, nil
}

func saveTrie(path string, trie *Trie, isIPv6 bool) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)

	// Write header
	header := Header{
		Version: config.IndexFormatVersion,
	}
	copy(header.Magic[:], Magic)
	if isIPv6 {
		header.Flags = FlagHasIPv6
	} else {
		header.Flags = FlagHasIPv4
	}

	if err := binary.Write(w, binary.LittleEndian, &header); err != nil {
		return err
	}

	// Serialize nodes
	if err := serializeNode(w, trie.Root); err != nil {
		return err
	}

	// Write node count at the end for verification
	if err := binary.Write(w, binary.LittleEndian, uint32(trie.Count)); err != nil {
		return err
	}

	return w.Flush()
}

func serializeNode(w io.Writer, node *TrieNode) error {
	if node == nil {
		// Write nil marker
		if err := binary.Write(w, binary.LittleEndian, uint8(0xFF)); err != nil {
			return err
		}
		return nil
	}

	// Write prefix length
	if err := binary.Write(w, binary.LittleEndian, uint8(node.PrefixLen)); err != nil {
		return err
	}

	// Write prefix bytes
	prefixBytes := (node.PrefixLen + 7) / 8
	if prefixBytes > 0 {
		if len(node.Prefix) < prefixBytes {
			// Pad with zeros
			padded := make([]byte, prefixBytes)
			copy(padded, node.Prefix)
			if _, err := w.Write(padded); err != nil {
				return err
			}
		} else {
			if _, err := w.Write(node.Prefix[:prefixBytes]); err != nil {
				return err
			}
		}
	}

	// Write has data flag
	hasData := node.Data != nil
	if err := binary.Write(w, binary.LittleEndian, hasData); err != nil {
		return err
	}

	// Write data if present
	if hasData {
		// Country code (2 bytes)
		var cc [2]byte
		if len(node.Data.CountryCode) >= 2 {
			cc[0] = node.Data.CountryCode[0]
			cc[1] = node.Data.CountryCode[1]
		}
		if _, err := w.Write(cc[:]); err != nil {
			return err
		}

		// Prefix string (length + data)
		prefixStr := []byte(node.Data.PrefixStr)
		if err := binary.Write(w, binary.LittleEndian, uint16(len(prefixStr))); err != nil {
			return err
		}
		if _, err := w.Write(prefixStr); err != nil {
			return err
		}
	}

	// Write children flags and recurse
	hasLeft := node.Children[0] != nil
	hasRight := node.Children[1] != nil
	if err := binary.Write(w, binary.LittleEndian, hasLeft); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, hasRight); err != nil {
		return err
	}

	if hasLeft {
		if err := serializeNode(w, node.Children[0]); err != nil {
			return err
		}
	}
	if hasRight {
		if err := serializeNode(w, node.Children[1]); err != nil {
			return err
		}
	}

	return nil
}

func loadTrie(path string, isIPv6 bool) (*Trie, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	r := bytes.NewReader(data)

	// Read header
	var header Header
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}

	// Validate magic
	if string(header.Magic[:]) != Magic {
		return nil, fmt.Errorf("invalid magic: %s", header.Magic)
	}

	// Validate version
	if header.Version != config.IndexFormatVersion {
		return nil, fmt.Errorf("unsupported index version %d (expected %d)", header.Version, config.IndexFormatVersion)
	}

	trie := NewTrie(isIPv6)

	// Deserialize nodes
	root, err := deserializeNode(r)
	if err != nil {
		return nil, fmt.Errorf("deserialize nodes: %w", err)
	}
	trie.Root = root

	// Read node count
	var count uint32
	if err := binary.Read(r, binary.LittleEndian, &count); err != nil {
		return nil, fmt.Errorf("read count: %w", err)
	}
	trie.Count = int(count)

	return trie, nil
}

func deserializeNode(r io.Reader) (*TrieNode, error) {
	// Read prefix length
	var prefixLen uint8
	if err := binary.Read(r, binary.LittleEndian, &prefixLen); err != nil {
		return nil, err
	}

	// Check for nil marker
	if prefixLen == 0xFF {
		return nil, nil
	}

	node := &TrieNode{
		PrefixLen: int(prefixLen),
	}

	// Read prefix bytes
	prefixBytes := (int(prefixLen) + 7) / 8
	if prefixBytes > 0 {
		node.Prefix = make([]byte, prefixBytes)
		if _, err := io.ReadFull(r, node.Prefix); err != nil {
			return nil, err
		}
	}

	// Read has data flag
	var hasData bool
	if err := binary.Read(r, binary.LittleEndian, &hasData); err != nil {
		return nil, err
	}

	if hasData {
		node.Data = &PrefixData{}

		// Read country code
		var cc [2]byte
		if _, err := io.ReadFull(r, cc[:]); err != nil {
			return nil, err
		}
		node.Data.CountryCode = string(cc[:])

		// Read prefix string
		var prefixStrLen uint16
		if err := binary.Read(r, binary.LittleEndian, &prefixStrLen); err != nil {
			return nil, err
		}
		prefixStr := make([]byte, prefixStrLen)
		if _, err := io.ReadFull(r, prefixStr); err != nil {
			return nil, err
		}
		node.Data.PrefixStr = string(prefixStr)
	}

	// Read children flags
	var hasLeft, hasRight bool
	if err := binary.Read(r, binary.LittleEndian, &hasLeft); err != nil {
		return nil, err
	}
	if err := binary.Read(r, binary.LittleEndian, &hasRight); err != nil {
		return nil, err
	}

	if hasLeft {
		left, err := deserializeNode(r)
		if err != nil {
			return nil, err
		}
		node.Children[0] = left
	}
	if hasRight {
		right, err := deserializeNode(r)
		if err != nil {
			return nil, err
		}
		node.Children[1] = right
	}

	return node, nil
}
