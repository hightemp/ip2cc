package ripestat

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// ASOverviewData is the response data from as-overview endpoint.
type ASOverviewData struct {
	Resource  string `json:"resource"`
	Type      string `json:"type"`
	Block     Block  `json:"block"`
	Holder    string `json:"holder"`
	Announced bool   `json:"announced"`
	QueryTime string `json:"query_time"`
}

// Block contains RIR block information.
type Block struct {
	Resource string `json:"resource"`
	Desc     string `json:"desc"`
	Name     string `json:"name"`
}

// ASOverviewResult contains the result of an AS overview query.
type ASOverviewResult struct {
	ASN       int
	Holder    string
	Announced bool
	BlockDesc string
}

// GetASOverview fetches holder information for an ASN.
func (c *Client) GetASOverview(ctx context.Context, asn int) (*ASOverviewResult, error) {
	params := url.Values{}
	params.Set("resource", fmt.Sprintf("AS%d", asn))

	resp, err := c.Get(ctx, "as-overview", params)
	if err != nil {
		return nil, fmt.Errorf("get as-overview for AS%d: %w", asn, err)
	}

	var data ASOverviewData
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, fmt.Errorf("decode as-overview data: %w", err)
	}

	return &ASOverviewResult{
		ASN:       asn,
		Holder:    data.Holder,
		Announced: data.Announced,
		BlockDesc: data.Block.Desc,
	}, nil
}
