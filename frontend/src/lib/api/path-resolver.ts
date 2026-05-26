/**
 * Maps a frontend-canonical path (e.g. `/jobs`, `jobs/12`,
 * `inventory/batch/move`) to the backend path, which always lives under
 * `/api/`. Idempotent for callers that already pass `/api/...`.
 */
export function resolveBackendPath(path: string): string {
	const stripped = path.replace(/^\/+/, '');
	if (stripped === '') return '/';
	if (stripped === 'api' || stripped.startsWith('api/')) return `/${stripped}`;
	return `/api/${stripped}`;
}
