// Package config provides configuration and path management.
package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppName is the application name.
	AppName = "ip2cc"

	// CacheDirName is the cache directory name.
	CacheDirName = ".ip2cc"

	// SnapshotsDirName is the snapshots subdirectory name.
	SnapshotsDirName = "snapshots"

	// LatestSymlink is the name of the latest snapshot symlink.
	LatestSymlink = "latest"

	// MetadataFileName is the metadata file name.
	MetadataFileName = "metadata.json"

	// IndexV4FileName is the IPv4 index file name.
	IndexV4FileName = "index_v4.bin"

	// IndexV6FileName is the IPv6 index file name.
	IndexV6FileName = "index_v6.bin"

	// RawDirName is the raw data directory name.
	RawDirName = "raw"

	// ProviderCacheFileName is the provider cache file name.
	ProviderCacheFileName = "provider_cache.json"

	// DefaultConcurrency is the default download concurrency.
	DefaultConcurrency = 8

	// MaxConcurrency is the maximum allowed concurrency (RIPEstat limit).
	MaxConcurrency = 8

	// DefaultProviderCacheTTLDays is the default TTL for provider cache.
	DefaultProviderCacheTTLDays = 7

	// DefaultProviderLookupConcurrency is the concurrency for provider lookups.
	DefaultProviderLookupConcurrency = 4

	// RIPEstatSourceApp is the sourceapp parameter for RIPEstat API.
	RIPEstatSourceApp = "ip2cc"

	// IndexFormatVersion is the current index format version.
	IndexFormatVersion uint32 = 1
)

// Config holds runtime configuration.
type Config struct {
	CacheDir    string
	Concurrency int
	Offline     bool
	JSONOutput  bool
	KeepRaw     bool
	Force       bool
	Time        string
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		CacheDir:    DefaultCacheDir(),
		Concurrency: DefaultConcurrency,
	}
}

// DefaultCacheDir returns the default cache directory path.
func DefaultCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to current directory
		home = "."
	}
	return filepath.Join(home, CacheDirName, "cache")
}

// SnapshotsDir returns the snapshots directory path.
func SnapshotsDir(cacheDir string) string {
	return filepath.Join(cacheDir, SnapshotsDirName)
}

// SnapshotDir returns the path for a specific snapshot.
func SnapshotDir(cacheDir, date string) string {
	return filepath.Join(SnapshotsDir(cacheDir), date)
}

// LatestSnapshotPath returns the path to the latest symlink.
func LatestSnapshotPath(cacheDir string) string {
	return filepath.Join(SnapshotsDir(cacheDir), LatestSymlink)
}

// MetadataPath returns the metadata file path for a snapshot.
func MetadataPath(snapshotDir string) string {
	return filepath.Join(snapshotDir, MetadataFileName)
}

// IndexV4Path returns the IPv4 index file path for a snapshot.
func IndexV4Path(snapshotDir string) string {
	return filepath.Join(snapshotDir, IndexV4FileName)
}

// IndexV6Path returns the IPv6 index file path for a snapshot.
func IndexV6Path(snapshotDir string) string {
	return filepath.Join(snapshotDir, IndexV6FileName)
}

// RawDir returns the raw data directory path for a snapshot.
func RawDir(snapshotDir string) string {
	return filepath.Join(snapshotDir, RawDirName)
}

// ProviderCachePath returns the provider cache file path.
func ProviderCachePath(cacheDir string) string {
	return filepath.Join(cacheDir, ProviderCacheFileName)
}

// EnsureDir creates a directory if it doesn't exist.
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
