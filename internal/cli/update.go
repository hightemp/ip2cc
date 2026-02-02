package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hightemp/ip2cc/internal/config"
	"github.com/hightemp/ip2cc/internal/countries"
	"github.com/hightemp/ip2cc/internal/index"
	"github.com/hightemp/ip2cc/internal/ripestat"
	"github.com/hightemp/ip2cc/internal/snapshot"
	"github.com/spf13/cobra"
)

var (
	concurrency   int
	countriesFile string
	keepRaw       bool
	force         bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Download and build IP prefix index",
	Long: `Downloads country resource lists from RIPEstat and builds a local
index for fast IP lookups.

Examples:
  ip2cc update                     # Build latest snapshot
  ip2cc update --time 2025-01-01   # Build snapshot for specific date
  ip2cc update --concurrency 4     # Limit parallel downloads`,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().IntVar(&concurrency, "concurrency", config.DefaultConcurrency, "parallel download limit (max 8)")
	updateCmd.Flags().StringVar(&countriesFile, "countries-file", "", "file with country codes (one per line)")
	updateCmd.Flags().BoolVar(&keepRaw, "keep-raw", false, "keep raw JSON responses")
	updateCmd.Flags().BoolVar(&force, "force", false, "rebuild even if snapshot exists")
	updateCmd.Flags().StringVar(&timeFlag, "time", "", "build snapshot for specific date (YYYY-MM-DD)")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Validate concurrency
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > config.MaxConcurrency {
		concurrency = config.MaxConcurrency
	}

	// Determine snapshot date
	snapshotDate := timeFlag
	if snapshotDate == "" {
		snapshotDate = time.Now().Format("2006-01-02")
	}

	// Check if snapshot already exists
	mgr := snapshot.NewManager(cacheDir)
	if !force && mgr.SnapshotExists(snapshotDate) {
		fmt.Printf("Snapshot for %s already exists. Use --force to rebuild.\n", snapshotDate)
		return nil
	}

	// Get country list
	var countryCodes []string
	if countriesFile != "" {
		content, err := os.ReadFile(countriesFile)
		if err != nil {
			return fmt.Errorf("read countries file: %w", err)
		}
		countryCodes, err = countries.LoadFromFile(string(content))
		if err != nil {
			return fmt.Errorf("parse countries file: %w", err)
		}
	} else {
		countryCodes = countries.AllCodesLower()
	}

	fmt.Printf("Building snapshot for %s with %d countries...\n", snapshotDate, len(countryCodes))

	// Create snapshot directory
	snapshotDir, err := mgr.CreateSnapshot(snapshotDate)
	if err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	// Create raw directory if needed
	if keepRaw {
		if err := config.EnsureDir(config.RawDir(snapshotDir)); err != nil {
			return fmt.Errorf("create raw dir: %w", err)
		}
	}

	// Download country resources
	client := ripestat.NewClient()
	results := make([]*ripestat.CountryResourceListResult, len(countryCodes))
	var mu sync.Mutex
	var completed int64
	var errors []string

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	startTime := time.Now()

	for i, cc := range countryCodes {
		wg.Add(1)
		go func(idx int, countryCode string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result, err := client.GetCountryResourceList(ctx, countryCode, timeFlag)

			mu.Lock()
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", countryCode, err))
			} else {
				results[idx] = result

				// Save raw JSON if requested
				if keepRaw {
					rawPath := filepath.Join(config.RawDir(snapshotDir), countryCode+".json")
					os.WriteFile(rawPath, result.RawJSON, 0644)
				}
			}

			count := atomic.AddInt64(&completed, 1)
			mu.Unlock()

			// Progress update
			if count%10 == 0 || count == int64(len(countryCodes)) {
				fmt.Printf("\rDownloading: %d/%d countries...", count, len(countryCodes))
			}
		}(i, cc)
	}

	wg.Wait()
	fmt.Println()

	if len(errors) > 0 {
		fmt.Printf("Warning: %d countries had errors:\n", len(errors))
		for _, e := range errors[:min(5, len(errors))] {
			fmt.Printf("  - %s\n", e)
		}
		if len(errors) > 5 {
			fmt.Printf("  ... and %d more\n", len(errors)-5)
		}
	}

	// Build indices
	fmt.Print("Building IPv4 index...")
	v4Trie := index.NewTrie(false)
	v4Count := 0
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, prefix := range result.IPv4 {
			if err := v4Trie.InsertCIDR(prefix, result.CountryCode); err == nil {
				v4Count++
			}
		}
	}
	fmt.Printf(" %d prefixes\n", v4Count)

	fmt.Print("Building IPv6 index...")
	v6Trie := index.NewTrie(true)
	v6Count := 0
	for _, result := range results {
		if result == nil {
			continue
		}
		for _, prefix := range result.IPv6 {
			if err := v6Trie.InsertCIDR(prefix, result.CountryCode); err == nil {
				v6Count++
			}
		}
	}
	fmt.Printf(" %d prefixes\n", v6Count)

	// Save indices
	fmt.Print("Saving indices...")
	if err := index.SaveIndex(
		config.IndexV4Path(snapshotDir),
		config.IndexV6Path(snapshotDir),
		v4Trie,
		v6Trie,
	); err != nil {
		return fmt.Errorf("save indices: %w", err)
	}
	fmt.Println(" done")

	// Determine actual query time from results
	actualQueryTime := snapshotDate
	for _, result := range results {
		if result != nil && result.QueryTime != "" {
			actualQueryTime = result.QueryTime
			break
		}
	}

	// Save metadata
	meta := snapshot.NewMetadata()
	meta.RequestedTime = snapshotDate
	meta.ActualQueryTime = actualQueryTime
	meta.CountriesCount = len(countryCodes)
	meta.Countries = countryCodes
	meta.PrefixesV4 = v4Count
	meta.PrefixesV6 = v6Count
	meta.IsLatest = true

	if err := meta.Save(config.MetadataPath(snapshotDir)); err != nil {
		return fmt.Errorf("save metadata: %w", err)
	}

	// Update latest symlink
	if err := mgr.SetLatest(snapshotDate); err != nil {
		fmt.Printf("Warning: could not update latest symlink: %v\n", err)
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nSnapshot built successfully in %v\n", elapsed.Round(time.Second))
	fmt.Printf("  Date: %s\n", snapshotDate)
	fmt.Printf("  IPv4 prefixes: %d\n", v4Count)
	fmt.Printf("  IPv6 prefixes: %d\n", v6Count)
	fmt.Printf("  Location: %s\n", snapshotDir)

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// saveJSON saves data as JSON to a file.
func saveJSON(path string, data interface{}) error {
	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, content, 0644)
}
