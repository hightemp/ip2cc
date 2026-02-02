// Package snapshot manages index snapshots.
package snapshot

import (
	"encoding/json"
	"os"
	"time"
)

// Metadata contains snapshot metadata.
type Metadata struct {
	Version            int       `json:"version"`
	CreatedAt          time.Time `json:"created_at"`
	RequestedTime      string    `json:"requested_time"`
	ActualQueryTime    string    `json:"actual_query_time"`
	CountriesCount     int       `json:"countries_count"`
	Countries          []string  `json:"countries"`
	PrefixesV4         int       `json:"prefixes_v4"`
	PrefixesV6         int       `json:"prefixes_v6"`
	IndexFormatVersion int       `json:"index_format_version"`
	Source             string    `json:"source"`
	IsLatest           bool      `json:"is_latest"`
}

// MetadataVersion is the current metadata format version.
const MetadataVersion = 1

// NewMetadata creates a new metadata instance.
func NewMetadata() *Metadata {
	return &Metadata{
		Version:            MetadataVersion,
		CreatedAt:          time.Now().UTC(),
		IndexFormatVersion: 1,
		Source:             "RIPEstat country-resource-list",
	}
}

// Save writes metadata to a file.
func (m *Metadata) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadMetadata loads metadata from a file.
func LoadMetadata(path string) (*Metadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	return &m, nil
}
