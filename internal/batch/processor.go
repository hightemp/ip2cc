// Package batch handles batch IP processing from stdin.
package batch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/netip"
	"strings"
	"sync"

	"github.com/hightemp/ip2cc/internal/countries"
	"github.com/hightemp/ip2cc/internal/index"
	"github.com/hightemp/ip2cc/internal/output"
	"github.com/hightemp/ip2cc/internal/provider"
	"github.com/hightemp/ip2cc/internal/snapshot"
)

// Processor handles batch IP lookups.
type Processor struct {
	v4Trie      *index.Trie
	v6Trie      *index.Trie
	resolver    *provider.Resolver
	meta        *snapshot.Metadata
	concurrency int
}

// NewProcessor creates a new batch processor.
func NewProcessor(v4, v6 *index.Trie, resolver *provider.Resolver, meta *snapshot.Metadata) *Processor {
	return &Processor{
		v4Trie:      v4,
		v6Trie:      v6,
		resolver:    resolver,
		meta:        meta,
		concurrency: 4,
	}
}

// ProcessInput reads IPs from input and writes results to output.
func (p *Processor) ProcessInput(ctx context.Context, r io.Reader, w io.Writer, jsonOutput bool) error {
	scanner := bufio.NewScanner(r)
	var results []*output.LookupResult

	if jsonOutput {
		// Collect all results for JSON array output
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			result := p.processIP(ctx, line)
			results = append(results, result)
		}

		batch := &output.BatchResult{Results: results}
		jsonStr, err := batch.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(w, jsonStr)
	} else {
		// Stream output line by line
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			result := p.processIP(ctx, line)
			fmt.Fprintln(w, result.FormatText())
		}
	}

	return scanner.Err()
}

// ProcessInputConcurrent processes IPs concurrently.
func (p *Processor) ProcessInputConcurrent(ctx context.Context, r io.Reader, w io.Writer, jsonOutput bool) error {
	scanner := bufio.NewScanner(r)
	var lines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	results := make([]*output.LookupResult, len(lines))
	var wg sync.WaitGroup
	sem := make(chan struct{}, p.concurrency)

	for i, line := range lines {
		wg.Add(1)
		go func(idx int, ip string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = p.processIP(ctx, ip)
		}(i, line)
	}

	wg.Wait()

	if jsonOutput {
		batch := &output.BatchResult{Results: results}
		jsonStr, err := batch.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Fprintln(w, jsonStr)
	} else {
		for _, result := range results {
			fmt.Fprintln(w, result.FormatText())
		}
	}

	return nil
}

func (p *Processor) processIP(ctx context.Context, ipStr string) *output.LookupResult {
	result := &output.LookupResult{
		IP:           ipStr,
		SnapshotTime: p.meta.RequestedTime,
		IndexBuiltAt: p.meta.CreatedAt,
	}

	// Parse IP
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		result.Error = fmt.Sprintf("invalid IP: %v", err)
		return result
	}

	// Select trie based on IP version
	var trie *index.Trie
	if ip.Is4() {
		trie = p.v4Trie
	} else {
		trie = p.v6Trie
	}

	// Lookup in trie
	data := trie.Lookup(ip)
	if data == nil {
		result.Error = "not found in index"
		return result
	}

	result.CountryCode = data.CountryCode
	result.CountryName = countries.GetName(data.CountryCode)
	result.Network = data.PrefixStr

	// Resolve provider if resolver is available
	if p.resolver != nil {
		provResult, _ := p.resolver.Resolve(ctx, ipStr, data.PrefixStr)
		result.Provider = provResult
	}

	return result
}
