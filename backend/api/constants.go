package api

// Pagination constants for API endpoints
const (
	// DefaultCardsPageSize is the default page size for card-heavy endpoints
	DefaultCardsPageSize = 50

	// MaxCardsPageSize is the maximum page size for card-heavy endpoints
	MaxCardsPageSize = 100
)

// Batch operation limits
const (
	// MaxBatchIDs is the maximum number of IDs in a batch move/delete operation
	MaxBatchIDs = 1000

	// MaxBatchItems is the maximum number of items in a batch create operation
	MaxBatchItems = 500
)

// Job constants
const (
	// DefaultJobRetentionDays is the default number of days to retain completed jobs
	DefaultJobRetentionDays = 30
)
