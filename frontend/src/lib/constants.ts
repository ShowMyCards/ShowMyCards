/**
 * Application-wide constants
 *
 * All magic numbers, timeout values, and configuration constants should be defined here.
 */

/**
 * Timeout values in milliseconds
 */
export const TIMEOUTS = {
	/** Duration to display notification before auto-dismiss */
	NOTIFICATION_DISMISS: 5000,
	/** Debounce delay for search input */
	SEARCH_DEBOUNCE: 300,
	/** Debounce delay for expression validation */
	VALIDATION_DEBOUNCE: 500,
	/** Interval for polling job status */
	POLLING_INTERVAL: 3000
} as const;

/**
 * Pagination configuration
 */
export const PAGINATION = {
	/** Default number of items per page */
	DEFAULT_PAGE_SIZE: 20,
	/** Maximum number of search results to display */
	MAX_SEARCH_RESULTS: 10
} as const;

/**
 * Job status values
 */
export const JOB_STATUS = {
	PENDING: 'pending',
	IN_PROGRESS: 'in_progress',
	COMPLETED: 'completed',
	FAILED: 'failed',
	CANCELLED: 'cancelled'
} as const;

export type ScryfallLanguage = { code: string; name: string };

export const SCRYFALL_LANGUAGES: readonly ScryfallLanguage[] = [
	{ code: 'en', name: 'English' },
	{ code: 'de', name: 'German' },
	{ code: 'es', name: 'Spanish' },
	{ code: 'fr', name: 'French' },
	{ code: 'it', name: 'Italian' },
	{ code: 'ja', name: 'Japanese' },
	{ code: 'ko', name: 'Korean' },
	{ code: 'pt', name: 'Portuguese' },
	{ code: 'ru', name: 'Russian' },
	{ code: 'zhs', name: 'Chinese (Simplified)' },
	{ code: 'zht', name: 'Chinese (Traditional)' }
] as const;
