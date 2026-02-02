// Package output handles output formatting.
package output

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hightemp/ip2cc/internal/provider"
)

// LookupResult contains the result of an IP lookup.
type LookupResult struct {
	IP           string           `json:"ip"`
	CountryCode  string           `json:"country_code"`
	CountryName  string           `json:"country_name"`
	Network      string           `json:"network"`
	Provider     *provider.Result `json:"provider,omitempty"`
	SnapshotTime string           `json:"snapshot_time"`
	IndexBuiltAt time.Time        `json:"index_built_at"`
	Error        string           `json:"error,omitempty"`
}

// FormatText formats result as tab-separated text.
func (r *LookupResult) FormatText() string {
	if r.Error != "" {
		return fmt.Sprintf("%s\t-\t-\t-\tERROR: %s", r.IP, r.Error)
	}

	providerStr := "unknown"
	if r.Provider != nil {
		providerStr = r.Provider.GetHolderString()
	}

	return fmt.Sprintf("%s\t%s\t%s\t%s\t%s",
		r.IP,
		r.CountryCode,
		r.CountryName,
		r.Network,
		providerStr,
	)
}

// FormatJSON formats result as JSON.
func (r *LookupResult) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// BatchResult contains results for batch processing.
type BatchResult struct {
	Results []*LookupResult
}

// FormatText formats batch results as text (one line per result).
func (b *BatchResult) FormatText() string {
	var lines []string
	for _, r := range b.Results {
		lines = append(lines, r.FormatText())
	}
	return strings.Join(lines, "\n")
}

// FormatJSON formats batch results as JSON array.
func (b *BatchResult) FormatJSON() (string, error) {
	data, err := json.MarshalIndent(b.Results, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// FormatError formats an error line for batch output.
func FormatError(ip string, err error) string {
	return fmt.Sprintf("%s\t-\t-\t-\tERROR: %s", ip, err.Error())
}
