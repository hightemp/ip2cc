// ip2cc is a CLI tool that looks up country and provider information for IP addresses.
package main

import (
	"github.com/hightemp/ip2cc/internal/cli"
)

// Build information (set via ldflags)
var (
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

func main() {
	cli.Version = version
	cli.Commit = commit
	cli.BuildTime = buildTime
	cli.Execute()
}
