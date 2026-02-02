package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/hightemp/ip2cc/internal/config"
	"github.com/hightemp/ip2cc/internal/ripestat"
)

// Mode represents the provider resolution mode.
type Mode string

const (
	// ModeBGP uses network-info + as-overview (default).
	ModeBGP Mode = "bgp"
	// ModeWhois uses whois API.
	ModeWhois Mode = "whois"
	// ModeOff disables provider lookup.
	ModeOff Mode = "off"
)

// ParseMode parses a mode string.
func ParseMode(s string) (Mode, error) {
	switch s {
	case "bgp", "":
		return ModeBGP, nil
	case "whois":
		return ModeWhois, nil
	case "off":
		return ModeOff, nil
	default:
		return "", fmt.Errorf("invalid provider mode: %s (use bgp, whois, or off)", s)
	}
}

// Result contains provider resolution result.
type Result struct {
	Mode    Mode     `json:"mode"`
	ASNs    []int    `json:"asns,omitempty"`
	Holders []string `json:"holders,omitempty"`
	Source  string   `json:"source"`
	Cached  bool     `json:"cached"`
	Error   string   `json:"error,omitempty"`
}

// Resolver resolves provider information for IP addresses.
type Resolver struct {
	client      *ripestat.Client
	cache       *Cache
	mode        Mode
	useCache    bool
	concurrency int
}

// NewResolver creates a new provider resolver.
func NewResolver(mode Mode, cacheDir string, useCache bool) *Resolver {
	var cache *Cache
	if useCache && mode != ModeOff {
		cache = NewCache(
			config.ProviderCachePath(cacheDir),
			config.DefaultProviderCacheTTLDays,
		)
		cache.Load()
	}

	return &Resolver{
		client:      ripestat.NewClient(),
		cache:       cache,
		mode:        mode,
		useCache:    useCache,
		concurrency: config.DefaultProviderLookupConcurrency,
	}
}

// Resolve resolves provider information for an IP.
func (r *Resolver) Resolve(ctx context.Context, ip string, matchedPrefix string) (*Result, error) {
	if r.mode == ModeOff {
		return &Result{
			Mode:   ModeOff,
			Source: "disabled",
		}, nil
	}

	switch r.mode {
	case ModeBGP:
		return r.resolveBGP(ctx, ip)
	case ModeWhois:
		return r.resolveWhois(ctx, matchedPrefix)
	default:
		return nil, fmt.Errorf("unknown mode: %s", r.mode)
	}
}

func (r *Resolver) resolveBGP(ctx context.Context, ip string) (*Result, error) {
	result := &Result{
		Mode:   ModeBGP,
		Source: "RIPEstat network-info + as-overview",
	}

	// Get network info
	netInfo, err := r.client.GetNetworkInfo(ctx, ip)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	if len(netInfo.ASNs) == 0 {
		result.Error = "no ASN found (not routed)"
		return result, nil
	}

	result.ASNs = netInfo.ASNs

	// Resolve holders for each ASN
	var mu sync.Mutex
	var wg sync.WaitGroup
	holders := make([]string, 0, len(netInfo.ASNs))
	allCached := true

	sem := make(chan struct{}, r.concurrency)

	for _, asn := range netInfo.ASNs {
		// Check cache first
		if r.cache != nil {
			if holder, ok := r.cache.Get(asn); ok {
				mu.Lock()
				holders = append(holders, holder)
				mu.Unlock()
				continue
			}
		}
		allCached = false

		wg.Add(1)
		go func(asn int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			overview, err := r.client.GetASOverview(ctx, asn)
			if err != nil {
				return
			}

			mu.Lock()
			holders = append(holders, overview.Holder)
			mu.Unlock()

			if r.cache != nil {
				r.cache.Set(asn, overview.Holder)
			}
		}(asn)
	}

	wg.Wait()
	result.Holders = holders
	result.Cached = allCached && len(holders) > 0

	return result, nil
}

func (r *Resolver) resolveWhois(ctx context.Context, prefix string) (*Result, error) {
	result := &Result{
		Mode:   ModeWhois,
		Source: "RIPEstat whois",
	}

	provider, err := r.client.GetProviderFromWhois(ctx, prefix)
	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	result.Holders = []string{provider}
	return result, nil
}

// SaveCache persists the cache to disk.
func (r *Resolver) SaveCache() error {
	if r.cache != nil {
		return r.cache.Save()
	}
	return nil
}

// GetHolderString returns a formatted holder string.
func (r *Result) GetHolderString() string {
	if len(r.Holders) == 0 {
		if r.Error != "" {
			return "unknown"
		}
		return "unknown"
	}
	return r.Holders[0]
}
