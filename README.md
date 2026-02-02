# ip2cc

IP to Country Code - A fast, cross-platform CLI tool for looking up country and provider information for IP addresses.

## Features

- **Fast offline lookups**: Uses a local Patricia trie index for O(k) prefix matching
- **IPv4 and IPv6 support**: Full support for both IP versions
- **Provider/ISP detection**: Identifies the network operator via BGP/ASN or WHOIS
- **Historical data**: Query data for specific dates using `--time` flag
- **Batch processing**: Process thousands of IPs via stdin
- **Cross-platform**: Linux, macOS, Windows (amd64, arm64)
- **Self-contained**: Single binary with no runtime dependencies

## Installation

### Unix (Linux/macOS)

```bash
curl -fsSL https://raw.githubusercontent.com/hightemp/ip2cc/main/scripts/install.sh | bash
```

### Windows (PowerShell)

```powershell
iwr -useb https://raw.githubusercontent.com/hightemp/ip2cc/main/scripts/install.ps1 | iex
```

### From source

```bash
go install github.com/hightemp/ip2cc/cmd/ip2cc@latest
```

## Quick Start

```bash
# Download the IP prefix database
ip2cc update

# Look up an IP address
ip2cc 8.8.8.8
```

## Usage

### Single IP Lookup

```bash
# Basic lookup
ip2cc 8.8.8.8
# Output: 8.8.8.8	US	United States	8.8.8.0/24	GOOGLE LLC

# JSON output
ip2cc --json 8.8.8.8

# Offline mode (no provider lookup)
ip2cc --offline 8.8.8.8

# Use historical snapshot
ip2cc --time 2025-01-01 8.8.8.8
```

### Batch Processing

```bash
# From file
cat ips.txt | ip2cc

# From command
echo -e "8.8.8.8\n1.1.1.1" | ip2cc

# JSON array output
cat ips.txt | ip2cc --json
```

### Update Database

```bash
# Download latest data (all countries)
ip2cc update

# Download for specific date
ip2cc update --time 2025-01-01

# Limit concurrency
ip2cc update --concurrency 4

# Keep raw JSON responses
ip2cc update --keep-raw

# Force rebuild existing snapshot
ip2cc update --force
```

### Provider Mode

```bash
# BGP mode (default) - uses network-info + as-overview APIs
ip2cc --provider-mode bgp 8.8.8.8

# WHOIS mode - uses whois API
ip2cc --provider-mode whois 8.8.8.8

# Off - disable provider lookup
ip2cc --provider-mode off 8.8.8.8
```

## Output Format

### Text (default)

Tab-separated values:
```
<ip>	<country_code>	<country_name>	<network>	<provider>
```

Example:
```
8.8.8.8	US	United States	8.8.8.0/24	GOOGLE LLC
```

### JSON

```json
{
  "ip": "8.8.8.8",
  "country_code": "US",
  "country_name": "United States",
  "network": "8.8.8.0/24",
  "provider": {
    "mode": "bgp",
    "asns": [15169],
    "holders": ["GOOGLE LLC"],
    "source": "RIPEstat network-info + as-overview",
    "cached": false
  },
  "snapshot_time": "2025-02-02",
  "index_built_at": "2025-02-02T10:00:00Z"
}
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 2 | Invalid input (bad IP format) |
| 3 | No snapshot available |
| 4 | IP not found in index |
| 5 | Provider lookup failed |

## Data Sources

This tool uses data from [RIPEstat Data API](https://stat.ripe.net/docs/02.data-api/):

- **Country prefixes**: `country-resource-list` endpoint
- **Network info (BGP)**: `network-info` endpoint
- **ASN holders**: `as-overview` endpoint
- **WHOIS data**: `whois` endpoint

### Important Note on Data Accuracy

The country/network data represents **IP address registration and delegation** from Regional Internet Registries (RIRs), not physical geolocation. An IP registered to one country may be used in another.

## Index Format

The binary index uses a Patricia trie structure for efficient longest-prefix-match queries:

- **Version**: 1
- **Magic**: `IP2CCIDX`
- **Complexity**: O(k) lookup where k = address bits (32 for IPv4, 128 for IPv6)
- **Storage**: `~/.ip2cc/cache/snapshots/<date>/`

### Snapshot Structure

```
~/.ip2cc/cache/
├── snapshots/
│   ├── 2025-02-02/
│   │   ├── metadata.json
│   │   ├── index_v4.bin
│   │   ├── index_v6.bin
│   │   └── raw/           # (optional)
│   └── latest -> 2025-02-02
└── provider_cache.json
```

## Configuration

### Cache Directory

Default: `~/.ip2cc/cache`

Override with `--cache-dir`:
```bash
ip2cc --cache-dir /custom/path update
```

### Provider Cache TTL

Default: 7 days

ASN-to-holder mappings are cached locally to reduce API calls.

## Development

### Build

```bash
go build -o ip2cc ./cmd/ip2cc/
```

### Test

```bash
go test -v ./...
```

### Release Build

```bash
VERSION=v1.0.0
go build -ldflags="-s -w -X main.version=${VERSION}" -o ip2cc ./cmd/ip2cc/
```

## License

MIT

## Credits

- Data provided by [RIPE NCC](https://www.ripe.net/) via [RIPEstat](https://stat.ripe.net/)
- Country codes from [ISO 3166](https://www.iso.org/iso-3166-country-codes.html)
