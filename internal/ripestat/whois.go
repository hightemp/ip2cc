package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// WhoisData is the response data from whois endpoint.
type WhoisData struct {
	Records    [][]WhoisRecord `json:"records"`
	IRRRecords [][]WhoisRecord `json:"irr_records"`
	Resource   string          `json:"resource"`
	QueryTime  string          `json:"query_time"`
}

// WhoisRecord represents a single whois record field.
type WhoisRecord struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Details string `json:"details_link,omitempty"`
}

// WhoisResult contains parsed whois information.
type WhoisResult struct {
	Resource    string
	OrgName     string
	Description string
	Country     string
	NetName     string
}

// GetWhois fetches whois information for a resource (IP or prefix).
func (c *Client) GetWhois(ctx context.Context, resource string) (*WhoisResult, error) {
	params := url.Values{}
	params.Set("resource", resource)

	resp, err := c.Get(ctx, "whois", params)
	if err != nil {
		return nil, fmt.Errorf("get whois for %s: %w", resource, err)
	}

	var data WhoisData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode whois data: %w", err)
	}

	result := &WhoisResult{
		Resource: resource,
	}

	// Parse records to extract relevant fields
	for _, recordSet := range data.Records {
		for _, record := range recordSet {
			key := strings.ToLower(record.Key)
			switch key {
			case "org-name", "orgname":
				if result.OrgName == "" {
					result.OrgName = record.Value
				}
			case "descr", "description":
				if result.Description == "" {
					result.Description = record.Value
				}
			case "country":
				if result.Country == "" {
					result.Country = record.Value
				}
			case "netname", "net-name":
				if result.NetName == "" {
					result.NetName = record.Value
				}
			}
		}
	}

	return result, nil
}

// GetProviderFromWhois attempts to determine provider name from whois data.
func (c *Client) GetProviderFromWhois(ctx context.Context, resource string) (string, error) {
	result, err := c.GetWhois(ctx, resource)
	if err != nil {
		return "", err
	}

	// Try different fields in order of preference
	if result.OrgName != "" {
		return result.OrgName, nil
	}
	if result.Description != "" {
		return result.Description, nil
	}
	if result.NetName != "" {
		return result.NetName, nil
	}

	return "", fmt.Errorf("no provider information found in whois for %s", resource)
}
