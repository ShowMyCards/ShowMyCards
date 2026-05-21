package services

import (
	"backend/models"
	"context"
	"fmt"
)

// jobsHref is the in-app path every job-derived banner links to.
const jobsHref = "/jobs"

// BannerService computes the set of UI banners that should currently be shown.
//
// # Why there is no provider registry
//
// Every banner today is derived from the Job table — in-progress syncs and
// failed syncs — so BannerService queries JobService directly. A pluggable
// provider registry ([]func(context.Context) []models.Banner) would be pure
// ceremony for a single source.
//
// Introduce a registry only when a banner needs a source that is NOT the Job
// table — for example a startup migration/upgrade status, or a disk-space
// check. At that point: extract the job logic in GetActive into a
// jobBannerProvider, define a provider type, and have GetActive fan out across
// registered providers. Until then, a new job-derived banner is simply another
// branch in GetActive.
type BannerService struct {
	jobService *JobService
}

// NewBannerService creates a new banner service.
func NewBannerService(jobService *JobService) *BannerService {
	return &BannerService{jobService: jobService}
}

// GetActive returns the banners that should be displayed right now.
func (s *BannerService) GetActive(ctx context.Context) ([]models.Banner, error) {
	banners := []models.Banner{}

	// In-progress syncs: one banner per active job type.
	active, err := s.jobService.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading active jobs: %w", err)
	}
	seen := make(map[models.JobType]bool)
	for _, job := range active {
		if seen[job.Type] {
			continue
		}
		seen[job.Type] = true
		if banner, ok := inProgressBanner(job.Type); ok {
			banners = append(banners, banner)
		}
	}

	// Failed syncs: one banner per type whose most recent job failed. A later
	// successful (or in-progress) run replaces it as the latest job, so the
	// banner clears itself.
	for _, jobType := range []models.JobType{models.JobTypeBulkDataImport, models.JobTypeSetDataImport} {
		last, err := s.jobService.GetLastJobByType(ctx, jobType)
		if err != nil {
			return nil, fmt.Errorf("loading last %s job: %w", jobType, err)
		}
		if last == nil || last.Status != models.JobStatusFailed {
			continue
		}
		if banner, ok := failedBanner(last); ok {
			banners = append(banners, banner)
		}
	}

	return banners, nil
}

// inProgressBanner builds the banner for an in-progress sync of the given type.
func inProgressBanner(jobType models.JobType) (models.Banner, bool) {
	var message string
	switch jobType {
	case models.JobTypeBulkDataImport:
		message = "Updating card data from Scryfall — this may take a few minutes. " +
			"Search works as normal; cards in your inventory and lists may show incomplete details until it finishes."
	case models.JobTypeSetDataImport:
		message = "Updating set data from Scryfall — set names and icons may be incomplete until it finishes."
	default:
		return models.Banner{}, false
	}

	return models.Banner{
		ID:          "sync:" + string(jobType),
		Severity:    models.BannerSeverityInfo,
		Message:     message,
		Dismissible: false,
		Link:        &models.BannerLink{Label: "View progress", Href: jobsHref},
	}, true
}

// failedBanner builds the banner for a job type whose most recent run failed.
func failedBanner(job *models.Job) (models.Banner, bool) {
	var message string
	switch job.Type {
	case models.JobTypeBulkDataImport:
		message = "The last card data sync from Scryfall failed — prices and card details may be out of date. Search still works normally."
	case models.JobTypeSetDataImport:
		message = "The last set data sync from Scryfall failed — some set names and icons may be missing or outdated."
	default:
		return models.Banner{}, false
	}

	return models.Banner{
		// The job ID makes the banner reappear after a *new* failure even if an
		// earlier one was dismissed.
		ID:          fmt.Sprintf("job-failed:%s:%d", job.Type, job.ID),
		Severity:    models.BannerSeverityWarning,
		Message:     message,
		Dismissible: true,
		Link:        &models.BannerLink{Label: "View jobs", Href: jobsHref},
	}, true
}
