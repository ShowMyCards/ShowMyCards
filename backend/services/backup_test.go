package services

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"backend/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// stubExporter is a test double for the Exporter interface. It marshals a fixed
// document so backup behaviour can be exercised without the api package (which
// would create an import cycle).
type stubExporter struct {
	doc map[string]any
	err error
}

func (s stubExporter) MarshalExport(_ context.Context) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	return json.MarshalIndent(s.doc, "", "  ")
}

func newSettingsServiceForBackup(t *testing.T) *SettingsService {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	if err := db.AutoMigrate(&models.Setting{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	return NewSettingsService(db)
}

// TestCreateBackup_ProducesValidGzippedJSON verifies the export runs in-process
// and is written as a gzip file that gunzips to the original JSON document.
func TestCreateBackup_ProducesValidGzippedJSON(t *testing.T) {
	dataDir := t.TempDir()
	exporter := stubExporter{doc: map[string]any{
		"version":     1,
		"exported_at": "2026-05-16T03:00:00Z",
		"inventory": []map[string]any{
			{"scryfall_id": "abc-123", "quantity": 3},
		},
	}}
	backupService := NewBackupService(exporter, newSettingsServiceForBackup(t), dataDir)

	path, err := backupService.CreateBackup(context.Background())
	if err != nil {
		t.Fatalf("CreateBackup failed: %v", err)
	}

	// File lives in the backups subdirectory of the data dir.
	if got, want := filepath.Dir(path), filepath.Join(dataDir, "backups"); got != want {
		t.Errorf("backup directory = %q, want %q", got, want)
	}

	name := filepath.Base(path)
	if !hasBackupName(name) {
		t.Errorf("backup filename %q does not match expected pattern", name)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open backup file: %v", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("backup is not valid gzip: %v", err)
	}
	defer gz.Close()

	raw, err := io.ReadAll(gz)
	if err != nil {
		t.Fatalf("failed to read gzipped data: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf("backup payload is not valid JSON: %v", err)
	}

	if v, ok := data["version"].(float64); !ok || v != 1 {
		t.Errorf("export version = %v, want 1", data["version"])
	}
	inv, ok := data["inventory"].([]any)
	if !ok || len(inv) != 1 {
		t.Fatalf("expected 1 inventory item, got %v", data["inventory"])
	}
}

// TestPruneBackups_KeepsNewestN verifies retention keeps the newest N files and
// removes older ones, and that lexical (timestamp) order maps to chronological.
func TestPruneBackups_KeepsNewestN(t *testing.T) {
	dataDir := t.TempDir()
	backupService := NewBackupService(stubExporter{}, newSettingsServiceForBackup(t), dataDir)

	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}

	// Names are chronological in lexical order. Create them out of order to
	// confirm pruning sorts before selecting.
	names := []string{
		"showmycards-backup-2026-01-03T030000Z.json.gz",
		"showmycards-backup-2026-01-01T030000Z.json.gz",
		"showmycards-backup-2026-01-05T030000Z.json.gz",
		"showmycards-backup-2026-01-02T030000Z.json.gz",
		"showmycards-backup-2026-01-04T030000Z.json.gz",
	}
	for _, n := range names {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to write fixture %q: %v", n, err)
		}
	}

	// A non-matching file must be ignored by pruning.
	if err := os.WriteFile(filepath.Join(dir, "unrelated.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write unrelated file: %v", err)
	}

	removed, err := backupService.PruneBackups(2)
	if err != nil {
		t.Fatalf("PruneBackups failed: %v", err)
	}
	if removed != 3 {
		t.Errorf("removed = %d, want 3", removed)
	}

	remaining, err := backupService.listBackupFiles()
	if err != nil {
		t.Fatalf("listBackupFiles failed: %v", err)
	}

	want := map[string]bool{
		"showmycards-backup-2026-01-04T030000Z.json.gz": true,
		"showmycards-backup-2026-01-05T030000Z.json.gz": true,
	}
	if len(remaining) != len(want) {
		t.Fatalf("remaining = %v, want the 2 newest", remaining)
	}
	for _, n := range remaining {
		if !want[n] {
			t.Errorf("unexpected file kept: %q", n)
		}
	}

	// The unrelated file must still exist.
	if _, err := os.Stat(filepath.Join(dir, "unrelated.txt")); err != nil {
		t.Errorf("unrelated file was removed: %v", err)
	}
}

// TestPruneBackups_BelowRetentionRemovesNothing verifies pruning is a no-op when
// the file count is at or below the retention limit.
func TestPruneBackups_BelowRetentionRemovesNothing(t *testing.T) {
	dataDir := t.TempDir()
	backupService := NewBackupService(stubExporter{}, newSettingsServiceForBackup(t), dataDir)

	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	for _, n := range []string{
		"showmycards-backup-2026-01-01T030000Z.json.gz",
		"showmycards-backup-2026-01-02T030000Z.json.gz",
	} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o644); err != nil {
			t.Fatalf("failed to write fixture: %v", err)
		}
	}

	removed, err := backupService.PruneBackups(5)
	if err != nil {
		t.Fatalf("PruneBackups failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

// TestPruneBackups_ZeroRetentionDisablesPruning verifies a retention of zero or
// less leaves all files untouched.
func TestPruneBackups_ZeroRetentionDisablesPruning(t *testing.T) {
	dataDir := t.TempDir()
	backupService := NewBackupService(stubExporter{}, newSettingsServiceForBackup(t), dataDir)

	dir := filepath.Join(dataDir, "backups")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("failed to create backup dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "showmycards-backup-2026-01-01T030000Z.json.gz"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}

	removed, err := backupService.PruneBackups(0)
	if err != nil {
		t.Fatalf("PruneBackups failed: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed = %d, want 0", removed)
	}
}

func hasBackupName(name string) bool {
	return len(name) > len(backupFilePrefix)+len(backupFileSuffix) &&
		name[:len(backupFilePrefix)] == backupFilePrefix &&
		name[len(name)-len(backupFileSuffix):] == backupFileSuffix
}
