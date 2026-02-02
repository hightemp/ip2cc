package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// CountryResourceListData is the response data from country-resource-list endpoint.
type CountryResourceListData struct {
	Resources struct {
		ASN  []string `json:"asn"`
		IPv4 []string `json:"ipv4"`
		IPv6 []string `json:"ipv6"`
	} `json:"resources"`
	QueryTime string `json:"query_time"`
	Resource  string `json:"resource"`
}

// CountryResourceListResult contains the result of a country resource list query.
type CountryResourceListResult struct {
	CountryCode string
	IPv4        []string
	IPv6        []string
	QueryTime   string
	RawJSON     []byte
}

// GetCountryResourceList fetches IPv4 and IPv6 prefixes for a country.
// countryCode should be lowercase ISO-3166 alpha-2 code.
// time is optional (format: YYYY-MM-DD or empty for latest).
func (c *Client) GetCountryResourceList(ctx context.Context, countryCode string, queryTime string) (*CountryResourceListResult, error) {
	params := url.Values{}
	params.Set("resource", strings.ToLower(countryCode))
	params.Set("v4_format", "prefix")

	if queryTime != "" {
		params.Set("time", queryTime)
	}

	resp, err := c.Get(ctx, "country-resource-list", params)
	if err != nil {
		return nil, fmt.Errorf("get country-resource-list for %s: %w", countryCode, err)
	}

	var data CountryResourceListData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode country-resource-list data: %w", err)
	}

	return &CountryResourceListResult{
		CountryCode: strings.ToUpper(countryCode),
		IPv4:        data.Resources.IPv4,
		IPv6:        data.Resources.IPv6,
		QueryTime:   data.QueryTime,
		RawJSON:     resp.Data,
	}, nil
}
