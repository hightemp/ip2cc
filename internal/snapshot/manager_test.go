package snapshot

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestManagerCreateSnapshot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	date := "2025-01-15"

	dir, err := mgr.CreateSnapshot(date)
	if err != nil {
		t.Fatalf("CreateSnapshot failed: %v", err)
	}

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("Snapshot directory not created")
	}

	expectedPath := filepath.Join(tmpDir, "snapshots", date)
	if dir != expectedPath {
		t.Errorf("Snapshot dir = %s, expected %s", dir, expectedPath)
	}
}

func TestManagerSnapshotExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	date := "2025-01-15"

	// Should not exist initially
	if mgr.SnapshotExists(date) {
		t.Error("Snapshot should not exist initially")
	}

	// Create snapshot directory and metadata
	dir, _ := mgr.CreateSnapshot(date)
	meta := NewMetadata()
	meta.RequestedTime = date
	meta.Save(filepath.Join(dir, "metadata.json"))

	// Should exist now
	if !mgr.SnapshotExists(date) {
		t.Error("Snapshot should exist after creation")
	}
}

func TestManagerListSnapshots(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	dates := []string{"2025-01-01", "2025-01-15", "2025-02-01"}

	for _, date := range dates {
		dir, _ := mgr.CreateSnapshot(date)
		meta := NewMetadata()
		meta.Save(filepath.Join(dir, "metadata.json"))
	}

	// List snapshots
	snapshots, err := mgr.ListSnapshots()
	if err != nil {
		t.Fatalf("ListSnapshots failed: %v", err)
	}

	if len(snapshots) != len(dates) {
		t.Errorf("Expected %d snapshots, got %d", len(dates), len(snapshots))
	}
}

func TestManagerSetLatest(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	date := "2025-01-15"

	dir, _ := mgr.CreateSnapshot(date)
	meta := NewMetadata()
	meta.RequestedTime = date
	meta.Save(filepath.Join(dir, "metadata.json"))

	// Set as latest
	if err := mgr.SetLatest(date); err != nil {
		t.Fatalf("SetLatest failed: %v", err)
	}

	// Get latest should return this snapshot
	latestDir, latestMeta, err := mgr.GetLatestSnapshot()
	if err != nil {
		t.Fatalf("GetLatestSnapshot failed: %v", err)
	}

	if latestDir != dir {
		t.Errorf("Latest dir = %s, expected %s", latestDir, dir)
	}

	if latestMeta.RequestedTime != date {
		t.Errorf("Latest meta date = %s, expected %s", latestMeta.RequestedTime, date)
	}
}

func TestManagerGetSnapshotByDate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	date := "2025-01-15"

	dir, _ := mgr.CreateSnapshot(date)
	meta := NewMetadata()
	meta.RequestedTime = date
	meta.CountriesCount = 100
	meta.Save(filepath.Join(dir, "metadata.json"))

	// Get by date
	foundDir, foundMeta, err := mgr.GetSnapshotByDate(date)
	if err != nil {
		t.Fatalf("GetSnapshotByDate failed: %v", err)
	}

	if foundDir != dir {
		t.Errorf("Dir = %s, expected %s", foundDir, dir)
	}

	if foundMeta.CountriesCount != 100 {
		t.Errorf("CountriesCount = %d, expected 100", foundMeta.CountriesCount)
	}
}

func TestManagerGetSnapshotByDateNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)

	_, _, err = mgr.GetSnapshotByDate("2025-01-15")
	if err == nil {
		t.Error("Expected error for nonexistent snapshot")
	}
}

func TestManagerDeleteSnapshot(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mgr := NewManager(tmpDir)
	date := "2025-01-15"

	dir, _ := mgr.CreateSnapshot(date)
	meta := NewMetadata()
	meta.Save(filepath.Join(dir, "metadata.json"))

	// Delete
	if err := mgr.DeleteSnapshot(date); err != nil {
		t.Fatalf("DeleteSnapshot failed: %v", err)
	}

	if mgr.SnapshotExists(date) {
		t.Error("Snapshot should not exist after deletion")
	}
}

func TestMetadataSaveAndLoad(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "ip2cc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	metaPath := filepath.Join(tmpDir, "metadata.json")

	// Create and save
	meta := NewMetadata()
	meta.RequestedTime = "2025-01-15"
	meta.ActualQueryTime = "2025-01-15T00:00:00Z"
	meta.CountriesCount = 249
	meta.Countries = []string{"us", "gb", "de"}
	meta.PrefixesV4 = 450000
	meta.PrefixesV6 = 120000
	meta.IsLatest = true

	if err := meta.Save(metaPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Load and verify
	loaded, err := LoadMetadata(metaPath)
	if err != nil {
		t.Fatalf("LoadMetadata failed: %v", err)
	}

	if loaded.RequestedTime != meta.RequestedTime {
		t.Errorf("RequestedTime = %s, expected %s", loaded.RequestedTime, meta.RequestedTime)
	}
	if loaded.CountriesCount != meta.CountriesCount {
		t.Errorf("CountriesCount = %d, expected %d", loaded.CountriesCount, meta.CountriesCount)
	}
	if loaded.PrefixesV4 != meta.PrefixesV4 {
		t.Errorf("PrefixesV4 = %d, expected %d", loaded.PrefixesV4, meta.PrefixesV4)
	}
	if loaded.PrefixesV6 != meta.PrefixesV6 {
		t.Errorf("PrefixesV6 = %d, expected %d", loaded.PrefixesV6, meta.PrefixesV6)
	}
}

func TestNewMetadata(t *testing.T) {
	meta := NewMetadata()

	if meta.Version != MetadataVersion {
		t.Errorf("Version = %d, expected %d", meta.Version, MetadataVersion)
	}

	if meta.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}

	if time.Since(meta.CreatedAt) > time.Minute {
		t.Error("CreatedAt should be recent")
	}

	if meta.Source == "" {
		t.Error("Source should be set")
	}
}
