import { describe, it, expect } from 'vitest';
import { resolveBackendPath, BACKEND_API_SEGMENTS } from './path-resolver';

describe('resolveBackendPath', () => {
	describe('segments hosted under /api/ on the backend', () => {
		it('prefixes a single-segment path', () => {
			expect(resolveBackendPath('/jobs')).toBe('/api/jobs');
		});

		it('prefixes a nested path', () => {
			expect(resolveBackendPath('/jobs/123')).toBe('/api/jobs/123');
		});

		it('preserves a query string', () => {
			expect(resolveBackendPath('/jobs?status=pending')).toBe('/api/jobs?status=pending');
		});

		it.each([...BACKEND_API_SEGMENTS])(
			'prefixes every advertised /api/ segment (%s)',
			(segment) => {
				expect(resolveBackendPath(`/${segment}`)).toBe(`/api/${segment}`);
			}
		);
	});

	describe('root-hosted segments', () => {
		it('leaves /inventory at the root', () => {
			expect(resolveBackendPath('/inventory')).toBe('/inventory');
		});

		it('leaves a nested path under a root segment', () => {
			expect(resolveBackendPath('/inventory/batch/move')).toBe('/inventory/batch/move');
		});

		it('leaves hyphenated root segments alone', () => {
			expect(resolveBackendPath('/sorting-rules/validate')).toBe('/sorting-rules/validate');
		});

		it('matches segments exactly, not by prefix', () => {
			// guard against someone "helpfully" changing `.has()` to a startsWith check
			expect(resolveBackendPath('/jobsmith')).toBe('/jobsmith');
		});
	});

	describe('leading-slash handling', () => {
		it('accepts a path with no leading slash', () => {
			expect(resolveBackendPath('jobs')).toBe('/api/jobs');
		});

		it('collapses multiple leading slashes', () => {
			expect(resolveBackendPath('//jobs')).toBe('/api/jobs');
		});

		it('returns "/" for an empty path', () => {
			expect(resolveBackendPath('')).toBe('/');
		});

		it('returns "/" for just "/"', () => {
			expect(resolveBackendPath('/')).toBe('/');
		});
	});

	describe('already-prefixed paths', () => {
		it('passes /api/... through without doubling the prefix', () => {
			// apiClient's server branch may receive paths that already include the
			// /api/ prefix — the resolver should treat the leading /api/ as already-done
			expect(resolveBackendPath('/api/data/import')).toBe('/api/data/import');
		});
	});
});
