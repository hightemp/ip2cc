// Package cli implements the command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/hightemp/ip2cc/internal/config"
	"github.com/spf13/cobra"
)

var (
	// Version information (set at build time)
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

// Global flags
var (
	cacheDir     string
	providerMode string
	offline      bool
	jsonOutput   bool
	timeFlag     string
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "ip2cc [ip]",
	Short: "IP to Country Code - lookup country and provider for IP addresses",
	Long: `ip2cc is a CLI tool that looks up country and provider information
for IP addresses using data from RIPEstat.

For single IP lookup:
  ip2cc 8.8.8.8

For batch processing (read from stdin):
  cat ips.txt | ip2cc

Data is derived from RIR (Regional Internet Registry) allocation data.
Note: This represents IP address registration/delegation, not physical geolocation.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLookup,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&cacheDir, "cache-dir", config.DefaultCacheDir(), "cache directory path")

	// Lookup-specific flags
	rootCmd.Flags().StringVar(&providerMode, "provider-mode", "bgp", "provider resolution mode: bgp, whois, or off")
	rootCmd.Flags().BoolVar(&offline, "offline", false, "offline mode (no network calls)")
	rootCmd.Flags().BoolVar(&jsonOutput, "json", false, "output in JSON format")
	rootCmd.Flags().StringVar(&timeFlag, "time", "", "use snapshot for specific date (YYYY-MM-DD)")

	// Add subcommands
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
}

// ExitCode constants
const (
	ExitSuccess        = 0
	ExitInvalidInput   = 2
	ExitNoSnapshot     = 3
	ExitNotFound       = 4
	ExitProviderFailed = 5
)

func exitWithCode(code int, msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(code)
}
