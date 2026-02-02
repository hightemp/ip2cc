package cli

import (
	"bufio"
	"context"
	"fmt"
	"net/netip"
	"os"

	"github.com/hightemp/ip2cc/internal/batch"
	"github.com/hightemp/ip2cc/internal/config"
	"github.com/hightemp/ip2cc/internal/countries"
	"github.com/hightemp/ip2cc/internal/index"
	"github.com/hightemp/ip2cc/internal/output"
	"github.com/hightemp/ip2cc/internal/provider"
	"github.com/hightemp/ip2cc/internal/snapshot"
	"github.com/spf13/cobra"
)

func runLookup(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Load snapshot
	mgr := snapshot.NewManager(cacheDir)
	var snapshotDir string
	var meta *snapshot.Metadata
	var err error

	if timeFlag != "" {
		snapshotDir, meta, err = mgr.GetSnapshotByDate(timeFlag)
	} else {
		snapshotDir, meta, err = mgr.GetLatestSnapshot()
	}

	if err != nil {
		exitWithCode(ExitNoSnapshot, fmt.Sprintf("Error: %v\nRun 'ip2cc update' to download data.", err))
		return nil
	}

	// Load indices
	v4Trie, v6Trie, err := index.LoadIndex(
		config.IndexV4Path(snapshotDir),
		config.IndexV6Path(snapshotDir),
	)
	if err != nil {
		exitWithCode(ExitNoSnapshot, fmt.Sprintf("Error loading index: %v", err))
		return nil
	}

	// Setup provider resolver
	var resolver *provider.Resolver
	if !offline {
		mode, err := provider.ParseMode(providerMode)
		if err != nil {
			exitWithCode(ExitInvalidInput, err.Error())
			return nil
		}
		resolver = provider.NewResolver(mode, cacheDir, true)
		defer resolver.SaveCache()
	}

	// Check if we have an IP argument or should read from stdin
	if len(args) == 1 {
		// Single IP lookup
		return lookupSingle(ctx, args[0], v4Trie, v6Trie, resolver, meta)
	}

	// Check if stdin is a terminal
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		// stdin is a terminal, show help
		return cmd.Help()
	}

	// Batch mode from stdin
	processor := batch.NewProcessor(v4Trie, v6Trie, resolver, meta)
	return processor.ProcessInput(ctx, os.Stdin, os.Stdout, jsonOutput)
}

func lookupSingle(ctx context.Context, ipStr string, v4, v6 *index.Trie, resolver *provider.Resolver, meta *snapshot.Metadata) error {
	result := &output.LookupResult{
		IP:           ipStr,
		SnapshotTime: meta.RequestedTime,
		IndexBuiltAt: meta.CreatedAt,
	}

	// Parse IP
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		exitWithCode(ExitInvalidInput, fmt.Sprintf("Invalid IP address: %s", ipStr))
		return nil
	}

	// Select trie based on IP version
	var trie *index.Trie
	if ip.Is4() {
		trie = v4
	} else {
		trie = v6
	}

	// Lookup in trie
	data := trie.Lookup(ip)
	if data == nil {
		exitWithCode(ExitNotFound, fmt.Sprintf("IP %s not found in index", ipStr))
		return nil
	}

	result.CountryCode = data.CountryCode
	result.CountryName = countries.GetName(data.CountryCode)
	result.Network = data.PrefixStr

	// Resolve provider
	if resolver != nil {
		provResult, _ := resolver.Resolve(ctx, ipStr, data.PrefixStr)
		result.Provider = provResult
	}

	// Output
	if jsonOutput {
		jsonStr, err := result.FormatJSON()
		if err != nil {
			return err
		}
		fmt.Println(jsonStr)
	} else {
		fmt.Println(result.FormatText())
	}

	return nil
}

// isBatchMode checks if we're receiving batch input
func isBatchMode() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// readBatchIPs reads IPs from stdin
func readBatchIPs() ([]string, error) {
	var ips []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			ips = append(ips, line)
		}
	}
	return ips, scanner.Err()
}
