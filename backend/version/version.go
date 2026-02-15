package version

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// Version is the application version, set at build time via ldflags:
//
//	go build -ldflags "-X backend/version.Version=1.2.3"
//
// Defaults to "dev" for local development.
var Version = "dev"

// SettingsStore is the subset of the settings service needed for version checks.
type SettingsStore interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

const settingsKey = "app_version"

// CheckAndUpdate reads the stored app version from the database and compares it
// to the running version. If the database was written by a newer version, it
// returns an error (the caller should exit). Otherwise it updates the stored
// version to the current one.
//
// The "dev" version is always considered compatible and never blocks startup.
func CheckAndUpdate(ctx context.Context, store SettingsStore) error {
	if Version == "dev" {
		return nil
	}

	stored, err := store.Get(ctx, settingsKey)
	if err != nil || stored == "" || stored == "dev" {
		// First run or previously ran dev — record current version
		return store.Set(ctx, settingsKey, Version)
	}

	cmp, err := Compare(stored, Version)
	if err != nil {
		// Can't parse one of the versions — update and continue
		return store.Set(ctx, settingsKey, Version)
	}

	if cmp > 0 {
		return fmt.Errorf(
			"database was last used by version %s, which is newer than the running version %s — "+
				"refusing to start to prevent data loss (upgrade the application or restore a compatible database)",
			stored, Version,
		)
	}

	// Stored version is older or equal — update to current
	return store.Set(ctx, settingsKey, Version)
}

// Compare compares two semver-style version strings (major.minor.patch).
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Versions may optionally have a "v" prefix.
func Compare(a, b string) (int, error) {
	av, err := parse(a)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", a, err)
	}
	bv, err := parse(b)
	if err != nil {
		return 0, fmt.Errorf("invalid version %q: %w", b, err)
	}

	for i := 0; i < 3; i++ {
		if av[i] < bv[i] {
			return -1, nil
		}
		if av[i] > bv[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parse(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) != 3 {
		return [3]int{}, fmt.Errorf("expected major.minor.patch format")
	}

	var result [3]int
	for i, p := range parts {
		// Strip any pre-release suffix (e.g. "1-rc1" -> "1")
		if idx := strings.IndexByte(p, '-'); idx >= 0 {
			p = p[:idx]
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return [3]int{}, fmt.Errorf("non-numeric component %q", parts[i])
		}
		result[i] = n
	}
	return result, nil
}
