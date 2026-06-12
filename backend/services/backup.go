package services

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	// DefaultBackupRetention is the default number of backup files to keep.
	DefaultBackupRetention = 14

	// backupFilePrefix and backupFileSuffix bound the timestamped backup
	// filenames, e.g. "showmycards-backup-2026-05-16T030000Z.json.gz".
	backupFilePrefix = "showmycards-backup-"
	backupFileSuffix = ".json.gz"

	// backupTimestampLayout matches the UTC timestamp embedded in filenames.
	// Colons are omitted so the names are valid on all filesystems.
	backupTimestampLayout = "2006-01-02T150405Z"
)

// Exporter produces the user-data export document in-process (no HTTP
// round-trip). It is implemented by the api.DataHandler so the backup job and
// the export endpoint share a single source of truth for the document shape.
type Exporter interface {
	MarshalExport(ctx context.Context) ([]byte, error)
}

// BackupService writes gzipped JSON snapshots of the user-owned data to disk
// and prunes old snapshots beyond a configurable retention limit. It reuses the
// in-process export logic so backups are byte-for-byte equivalent to a manual
// export download.
type BackupService struct {
	exporter        Exporter
	settingsService *SettingsService
	dataDir         string
}

// NewBackupService creates a new backup service. dataDir is the base data
// directory (derived from configuration, e.g. the DATA_DIR env var); backups
// are written to the "backups" subdirectory within it.
func NewBackupService(exporter Exporter, settingsService *SettingsService, dataDir string) *BackupService {
	return &BackupService{
		exporter:        exporter,
		settingsService: settingsService,
		dataDir:         dataDir,
	}
}

// backupDir returns the directory backups are written to.
func (s *BackupService) backupDir() string {
	return filepath.Join(s.dataDir, "backups")
}

// CreateBackup builds the export, gzips it, and writes a timestamped backup
// file. It returns the path of the file written.
func (s *BackupService) CreateBackup(ctx context.Context) (string, error) {
	jsonData, err := s.exporter.MarshalExport(ctx)
	if err != nil {
		return "", fmt.Errorf("build export: %w", err)
	}

	dir := s.backupDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create backup directory: %w", err)
	}

	filename := backupFilePrefix + time.Now().UTC().Format(backupTimestampLayout) + backupFileSuffix
	path := filepath.Join(dir, filename)

	// Write to a temp file first, then rename, so a crash mid-write never
	// leaves a truncated backup behind.
	tmp, err := os.CreateTemp(dir, filename+".tmp-*")
	if err != nil {
		return "", fmt.Errorf("create temp backup file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // no-op once the rename succeeds

	gz := gzip.NewWriter(tmp)
	if _, err := gz.Write(jsonData); err != nil {
		gz.Close()
		tmp.Close()
		return "", fmt.Errorf("write gzip data: %w", err)
	}
	if err := gz.Close(); err != nil {
		tmp.Close()
		return "", fmt.Errorf("flush gzip data: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("close temp backup file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return "", fmt.Errorf("finalize backup file: %w", err)
	}

	return path, nil
}

// PruneBackups removes the oldest backup files so that at most retention files
// remain. A retention of zero or less disables pruning. It returns the number
// of files removed.
func (s *BackupService) PruneBackups(retention int) (int, error) {
	if retention <= 0 {
		return 0, nil
	}

	files, err := s.listBackupFiles()
	if err != nil {
		return 0, err
	}

	if len(files) <= retention {
		return 0, nil
	}

	// Filenames embed a sortable UTC timestamp, so lexical order is
	// chronological. Oldest entries sort first and are the ones to remove.
	sort.Strings(files)

	removed := 0
	dir := s.backupDir()
	for _, name := range files[:len(files)-retention] {
		if err := os.Remove(filepath.Join(dir, name)); err != nil {
			return removed, fmt.Errorf("remove old backup %q: %w", name, err)
		}
		removed++
	}

	return removed, nil
}

// listBackupFiles returns the names of backup files in the backup directory.
// A missing directory is treated as an empty set rather than an error.
func (s *BackupService) listBackupFiles() ([]string, error) {
	entries, err := os.ReadDir(s.backupDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read backup directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, backupFilePrefix) && strings.HasSuffix(name, backupFileSuffix) {
			names = append(names, name)
		}
	}
	return names, nil
}
