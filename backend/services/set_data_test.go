package services

import (
	"backend/models"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	scryfall "github.com/BlueMonday/go-scryfall"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type stubSetLister struct {
	sets []scryfall.Set
	err  error
}

func (s *stubSetLister) ListSets(ctx context.Context) ([]scryfall.Set, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.sets, nil
}

func setupSetDataServiceTest(t *testing.T, lister SetLister) (*SetDataService, *JobService, *SettingsService, *gorm.DB, string) {
	t.Helper()

	// TranslateError: true matches database.NewClient so SQLite unique constraint
	// violations surface as gorm.ErrDuplicatedKey ("duplicated key not allowed").
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	if err := db.AutoMigrate(&models.Job{}, &models.Setting{}, &models.Set{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	dataDir := t.TempDir()
	jobService := NewJobService(db)
	settingsService := NewSettingsService(db)
	service := NewSetDataService(db, jobService, settingsService, lister, dataDir)

	return service, jobService, settingsService, db, dataDir
}

func runSetImport(t *testing.T, service *SetDataService) (*models.Job, error) {
	t.Helper()
	ctx := context.Background()
	job, err := service.CreateImportJob(ctx)
	if err != nil {
		t.Fatalf("CreateImportJob failed: %v", err)
	}
	importErr := service.DownloadAndImport(ctx, job.ID)
	updatedJob, err := service.jobService.Get(ctx, job.ID)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	return updatedJob, importErr
}

func TestSetDataService_DownloadAndImport_FreshImport(t *testing.T) {
	iconServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/svg+xml")
		_, _ = w.Write([]byte("<svg></svg>"))
	}))
	defer iconServer.Close()

	lister := &stubSetLister{
		sets: []scryfall.Set{
			{ID: "set-1", Code: "lea", Name: "Limited Edition Alpha", SetType: scryfall.SetTypeCore, CardCount: 295, IconSVGURI: iconServer.URL + "/lea.svg"},
			{ID: "set-2", Code: "leb", Name: "Limited Edition Beta", SetType: scryfall.SetTypeCore, CardCount: 302, IconSVGURI: iconServer.URL + "/leb.svg"},
		},
	}

	service, _, settingsService, db, dataDir := setupSetDataServiceTest(t, lister)

	job, err := runSetImport(t, service)
	if err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}
	if job.Status != models.JobStatusCompleted {
		t.Errorf("job status = %q, want %q", job.Status, models.JobStatusCompleted)
	}

	var sets []models.Set
	if err := db.Order("code ASC").Find(&sets).Error; err != nil {
		t.Fatalf("query sets: %v", err)
	}
	if len(sets) != 2 {
		t.Fatalf("expected 2 sets, got %d", len(sets))
	}
	if sets[0].ScryfallID != "set-1" || sets[0].Code != "lea" || sets[0].IconFilename != "lea.svg" {
		t.Errorf("unexpected first set: %+v", sets[0])
	}
	if _, err := os.Stat(filepath.Join(dataDir, "set-icons", "lea.svg")); err != nil {
		t.Errorf("expected icon file lea.svg to exist: %v", err)
	}

	status, err := settingsService.Get(context.Background(), "set_data_update_last_status")
	if err != nil {
		t.Fatalf("get last status: %v", err)
	}
	if status != "success" {
		t.Errorf("set_data_update_last_status = %q, want %q", status, "success")
	}
}

func TestSetDataService_DownloadAndImport_CodeReassigned(t *testing.T) {
	tests := []struct {
		name     string
		existing []models.Set
		incoming []scryfall.Set
		want     map[string]string // scryfall_id -> code
	}{
		{
			name: "code renamed on existing set and reused by new set listed first",
			existing: []models.Set{
				{ScryfallID: "id-old", Code: "punk", Name: "Black Lotus Unknown Planechase"},
			},
			// Scryfall lists newest sets first, so the new set claiming "punk"
			// arrives before the older set renamed from "punk" to "ounk".
			incoming: []scryfall.Set{
				{ID: "id-new", Code: "punk", Name: "Prismatic Pride"},
				{ID: "id-old", Code: "ounk", Name: "Black Lotus Unknown Planechase"},
			},
			want: map[string]string{
				"id-new": "punk",
				"id-old": "ounk",
			},
		},
		{
			name: "removed set code reused by new set",
			existing: []models.Set{
				{ScryfallID: "id-removed", Code: "om2", Name: "Through the Omenpaths 2"},
			},
			incoming: []scryfall.Set{
				{ID: "id-replacement", Code: "om2", Name: "Through the Omenpaths 2 (New)"},
			},
			want: map[string]string{
				"id-replacement": "om2",
			},
		},
		{
			name: "two existing sets swap codes",
			existing: []models.Set{
				{ScryfallID: "id-a", Code: "aaa", Name: "Set A"},
				{ScryfallID: "id-b", Code: "bbb", Name: "Set B"},
			},
			incoming: []scryfall.Set{
				{ID: "id-a", Code: "bbb", Name: "Set A"},
				{ID: "id-b", Code: "aaa", Name: "Set B"},
			},
			want: map[string]string{
				"id-a": "bbb",
				"id-b": "aaa",
			},
		},
		{
			name: "set removed from Scryfall catalog is pruned",
			existing: []models.Set{
				{ScryfallID: "id-keep", Code: "lea", Name: "Alpha"},
				{ScryfallID: "id-gone", Code: "om2", Name: "Removed Set"},
			},
			incoming: []scryfall.Set{
				{ID: "id-keep", Code: "lea", Name: "Alpha"},
			},
			want: map[string]string{
				"id-keep": "lea",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			lister := &stubSetLister{sets: tc.incoming}
			service, _, _, db, _ := setupSetDataServiceTest(t, lister)

			if err := db.Create(&tc.existing).Error; err != nil {
				t.Fatalf("seed existing sets: %v", err)
			}

			job, err := runSetImport(t, service)
			if err != nil {
				t.Fatalf("DownloadAndImport failed: %v", err)
			}
			if job.Status != models.JobStatusCompleted {
				t.Errorf("job status = %q, want %q", job.Status, models.JobStatusCompleted)
			}

			var got []models.Set
			if err := db.Find(&got).Error; err != nil {
				t.Fatalf("query sets: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %d sets (%+v), want %d", len(got), got, len(tc.want))
			}
			for _, s := range got {
				wantCode, ok := tc.want[s.ScryfallID]
				if !ok {
					t.Errorf("unexpected set in db: %+v", s)
					continue
				}
				if s.Code != wantCode {
					t.Errorf("set %s code = %q, want %q", s.ScryfallID, s.Code, wantCode)
				}
			}
		})
	}
}

func TestSetDataService_DownloadAndImport_EmptyCatalogPreservesExistingSets(t *testing.T) {
	lister := &stubSetLister{sets: []scryfall.Set{}}
	service, _, _, db, _ := setupSetDataServiceTest(t, lister)

	seed := models.Set{ScryfallID: "id-keep", Code: "lea", Name: "Alpha"}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed set: %v", err)
	}

	if _, err := runSetImport(t, service); err != nil {
		t.Fatalf("DownloadAndImport failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.Set{}).Count(&count).Error; err != nil {
		t.Fatalf("count sets: %v", err)
	}
	if count != 1 {
		t.Errorf("expected existing set to be preserved on empty catalog, got count %d", count)
	}
}

func TestSetDataService_DownloadAndImport_RollsBackOnSaveError(t *testing.T) {
	// Duplicate code inside the incoming catalog itself triggers a unique constraint
	// violation; the transaction must roll back so existing sets are not lost.
	lister := &stubSetLister{
		sets: []scryfall.Set{
			{ID: "id-1", Code: "dup", Name: "First"},
			{ID: "id-2", Code: "dup", Name: "Second"},
		},
	}
	service, _, settingsService, db, _ := setupSetDataServiceTest(t, lister)

	seed := models.Set{ScryfallID: "id-existing", Code: "lea", Name: "Alpha"}
	if err := db.Create(&seed).Error; err != nil {
		t.Fatalf("seed set: %v", err)
	}

	job, err := runSetImport(t, service)
	if !errors.Is(err, gorm.ErrDuplicatedKey) {
		t.Fatalf("expected gorm.ErrDuplicatedKey, got: %v", err)
	}
	if job.Status != models.JobStatusFailed {
		t.Errorf("job status = %q, want %q", job.Status, models.JobStatusFailed)
	}

	var got []models.Set
	if err := db.Find(&got).Error; err != nil {
		t.Fatalf("query sets: %v", err)
	}
	if len(got) != 1 || got[0].ScryfallID != "id-existing" {
		t.Errorf("expected existing set to remain intact after rollback, got %+v", got)
	}

	status, err := settingsService.Get(context.Background(), "set_data_update_last_status")
	if err != nil {
		t.Fatalf("get last status: %v", err)
	}
	if status != "failed" {
		t.Errorf("set_data_update_last_status = %q, want %q", status, "failed")
	}
}
