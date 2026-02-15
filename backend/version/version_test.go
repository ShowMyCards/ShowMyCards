package version

import (
	"context"
	"errors"
	"testing"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"0.1.0", "0.0.9", 1},
		{"v1.2.3", "1.2.3", 0},
		{"1.2.3-rc1", "1.2.3", 0},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := Compare(tt.a, tt.b)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCompare_Invalid(t *testing.T) {
	tests := []struct {
		a, b string
	}{
		{"not-a-version", "1.0.0"},
		{"1.0.0", "bad"},
		{"1.0", "1.0.0"},
		{"", "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			_, err := Compare(tt.a, tt.b)
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

// mockSettings is a simple in-memory settings store for testing.
type mockSettings struct {
	data map[string]string
}

func (m *mockSettings) Get(_ context.Context, key string) (string, error) {
	v, ok := m.data[key]
	if !ok {
		return "", errors.New("not found")
	}
	return v, nil
}

func (m *mockSettings) Set(_ context.Context, key, value string) error {
	m.data[key] = value
	return nil
}

func TestCheckAndUpdate_DevAlwaysPasses(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "dev"
	store := &mockSettings{data: map[string]string{"app_version": "99.0.0"}}

	if err := CheckAndUpdate(context.Background(), store); err != nil {
		t.Fatalf("dev version should never block: %v", err)
	}
}

func TestCheckAndUpdate_FirstRun(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "1.0.0"
	store := &mockSettings{data: map[string]string{}}

	if err := CheckAndUpdate(context.Background(), store); err != nil {
		t.Fatalf("first run should succeed: %v", err)
	}

	if store.data["app_version"] != "1.0.0" {
		t.Errorf("expected stored version 1.0.0, got %q", store.data["app_version"])
	}
}

func TestCheckAndUpdate_Upgrade(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "2.0.0"
	store := &mockSettings{data: map[string]string{"app_version": "1.5.0"}}

	if err := CheckAndUpdate(context.Background(), store); err != nil {
		t.Fatalf("upgrade should succeed: %v", err)
	}

	if store.data["app_version"] != "2.0.0" {
		t.Errorf("expected stored version 2.0.0, got %q", store.data["app_version"])
	}
}

func TestCheckAndUpdate_SameVersion(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "1.5.0"
	store := &mockSettings{data: map[string]string{"app_version": "1.5.0"}}

	if err := CheckAndUpdate(context.Background(), store); err != nil {
		t.Fatalf("same version should succeed: %v", err)
	}
}

func TestCheckAndUpdate_Downgrade_Blocked(t *testing.T) {
	original := Version
	defer func() { Version = original }()

	Version = "1.0.0"
	store := &mockSettings{data: map[string]string{"app_version": "2.0.0"}}

	err := CheckAndUpdate(context.Background(), store)
	if err == nil {
		t.Fatal("downgrade should be blocked")
	}
}
