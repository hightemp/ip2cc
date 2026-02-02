package snapshot

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hightemp/ip2cc/internal/config"
)

// Manager handles snapshot operations.
type Manager struct {
	cacheDir string
}

// NewManager creates a new snapshot manager.
func NewManager(cacheDir string) *Manager {
	return &Manager{cacheDir: cacheDir}
}

// GetSnapshotDir returns the directory for a specific date.
func (m *Manager) GetSnapshotDir(date string) string {
	return config.SnapshotDir(m.cacheDir, date)
}

// CreateSnapshot creates a new snapshot directory.
func (m *Manager) CreateSnapshot(date string) (string, error) {
	dir := m.GetSnapshotDir(date)
	if err := config.EnsureDir(dir); err != nil {
		return "", fmt.Errorf("create snapshot dir: %w", err)
	}
	return dir, nil
}

// SnapshotExists checks if a snapshot exists for the given date.
func (m *Manager) SnapshotExists(date string) bool {
	dir := m.GetSnapshotDir(date)
	metaPath := config.MetadataPath(dir)
	_, err := os.Stat(metaPath)
	return err == nil
}

// GetLatestSnapshot returns the latest snapshot directory and metadata.
func (m *Manager) GetLatestSnapshot() (string, *Metadata, error) {
	// First try the latest symlink
	latestPath := config.LatestSnapshotPath(m.cacheDir)
	target, err := os.Readlink(latestPath)
	if err == nil {
		// Symlink exists, resolve it
		if !filepath.IsAbs(target) {
			target = filepath.Join(config.SnapshotsDir(m.cacheDir), target)
		}
		meta, err := LoadMetadata(config.MetadataPath(target))
		if err == nil {
			return target, meta, nil
		}
	}

	// Fallback: find the most recent snapshot by date
	snapshots, err := m.ListSnapshots()
	if err != nil {
		return "", nil, err
	}
	if len(snapshots) == 0 {
		return "", nil, fmt.Errorf("no snapshots available")
	}

	// Sort by date descending
	sort.Sort(sort.Reverse(sort.StringSlice(snapshots)))
	latestDate := snapshots[0]
	dir := m.GetSnapshotDir(latestDate)
	meta, err := LoadMetadata(config.MetadataPath(dir))
	if err != nil {
		return "", nil, fmt.Errorf("load metadata for %s: %w", latestDate, err)
	}

	return dir, meta, nil
}

// GetSnapshotByDate returns snapshot for a specific date.
func (m *Manager) GetSnapshotByDate(date string) (string, *Metadata, error) {
	dir := m.GetSnapshotDir(date)
	metaPath := config.MetadataPath(dir)

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("snapshot for %s not found, run: ip2cc update --time %s", date, date)
	}

	meta, err := LoadMetadata(metaPath)
	if err != nil {
		return "", nil, fmt.Errorf("load metadata: %w", err)
	}

	return dir, meta, nil
}

// ListSnapshots returns all available snapshot dates.
func (m *Manager) ListSnapshots() ([]string, error) {
	snapshotsDir := config.SnapshotsDir(m.cacheDir)
	if err := config.EnsureDir(snapshotsDir); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(snapshotsDir)
	if err != nil {
		return nil, err
	}

	var dates []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		// Skip the latest symlink
		if name == config.LatestSymlink {
			continue
		}
		// Check if it's a valid date format (YYYY-MM-DD)
		if len(name) == 10 && strings.Count(name, "-") == 2 {
			dates = append(dates, name)
		}
	}

	return dates, nil
}

// SetLatest updates the latest symlink to point to the given date.
func (m *Manager) SetLatest(date string) error {
	latestPath := config.LatestSnapshotPath(m.cacheDir)

	// Remove existing symlink
	os.Remove(latestPath)

	// Create new symlink (relative path)
	return os.Symlink(date, latestPath)
}

// DeleteSnapshot removes a snapshot.
func (m *Manager) DeleteSnapshot(date string) error {
	dir := m.GetSnapshotDir(date)
	return os.RemoveAll(dir)
}
