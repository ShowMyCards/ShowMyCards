import { describe, it, expect } from 'vitest';
import { resolveBackendPath } from './path-resolver';

describe('resolveBackendPath', () => {
	describe('canonical paths', () => {
		it('prefixes a single-segment path', () => {
			expect(resolveBackendPath('/jobs')).toBe('/api/jobs');
		});

		it('prefixes a nested path', () => {
			expect(resolveBackendPath('/jobs/123')).toBe('/api/jobs/123');
		});

		it('preserves a query string', () => {
			expect(resolveBackendPath('/jobs?status=pending')).toBe('/api/jobs?status=pending');
		});

		it('prefixes hyphenated segments', () => {
			expect(resolveBackendPath('/sorting-rules/validate')).toBe('/api/sorting-rules/validate');
		});

		it('prefixes nested batch paths', () => {
			expect(resolveBackendPath('/inventory/batch/move')).toBe('/api/inventory/batch/move');
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

		it('handles a bare /api', () => {
			expect(resolveBackendPath('/api')).toBe('/api');
		});

		it('does not double-prefix a segment that merely starts with "api"', () => {
			expect(resolveBackendPath('/apiary')).toBe('/api/apiary');
		});
	});
});
