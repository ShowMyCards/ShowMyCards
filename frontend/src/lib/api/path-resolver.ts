/**
 * The backend exposes some routes under `/api/...` (jobs, data, settings,
 * etc.) and others at the root (`/inventory/...`, `/sorting-rules/...`).
 * Callers (apiClient, the catch-all proxy, direct fetches) treat the
 * frontend `/api/<resource>` URL as the canonical form. This resolver maps
 * that to whichever shape the backend actually uses, so call-site paths stay
 * consistent regardless of which side of the prefix split they fall on.
 */

export const BACKEND_API_SEGMENTS = new Set([
	'banners',
	'bulk-data',
	'dashboard',
	'data',
	'jobs',
	'scheduler',
	'settings'
]);

/**
 * Resolves a frontend-canonical path (e.g. `/jobs`, `jobs/12`,
 * `inventory/batch/move`) to the actual backend path (`/api/jobs`,
 * `/api/jobs/12`, `/inventory/batch/move`).
 */
export function resolveBackendPath(path: string): string {
	const stripped = path.replace(/^\/+/, '');
	const firstSegment = stripped.split('/')[0]?.split('?')[0] ?? '';
	const prefix = BACKEND_API_SEGMENTS.has(firstSegment) ? '/api/' : '/';
	return `${prefix}${stripped}`;
}
