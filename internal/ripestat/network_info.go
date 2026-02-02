package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// NetworkInfoData is the response data from network-info endpoint.
type NetworkInfoData struct {
	ASNs   []int  `json:"asns"`
	Prefix string `json:"prefix"`
}

// NetworkInfoResult contains the result of a network info query.
type NetworkInfoResult struct {
	IP     string
	Prefix string
	ASNs   []int
}

// GetNetworkInfo fetches BGP routing information for an IP address.
func (c *Client) GetNetworkInfo(ctx context.Context, ip string) (*NetworkInfoResult, error) {
	params := url.Values{}
	params.Set("resource", ip)

	resp, err := c.Get(ctx, "network-info", params)
	if err != nil {
		return nil, fmt.Errorf("get network-info for %s: %w", ip, err)
	}

	var data NetworkInfoData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode network-info data: %w", err)
	}

	return &NetworkInfoResult{
		IP:     ip,
		Prefix: data.Prefix,
		ASNs:   data.ASNs,
	}, nil
}
