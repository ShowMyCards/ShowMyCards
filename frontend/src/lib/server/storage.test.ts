import { describe, it, expect, vi, type Mock } from 'vitest';
import { loadStorageLocations, loadStorageLocationsWithCounts } from './storage';

const BACKEND_URL = 'http://localhost:3000';

function jsonResponse(body: unknown, status = 200): Response {
	return new Response(JSON.stringify(body), {
		status,
		headers: { 'content-type': 'application/json' }
	});
}

function asFetch(fetchMock: Mock): typeof globalThis.fetch {
	return fetchMock as unknown as typeof globalThis.fetch;
}

function loc(id: number) {
	return { id, name: `Box ${id}`, storage_type: 'Box' };
}

describe('loadStorageLocations', () => {
	it('requests the maximum page size', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValue(jsonResponse({ data: [loc(1)], total_pages: 1, page: 1 }));

		await loadStorageLocations(asFetch(fetchMock));

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock.mock.calls[0][0]).toBe(`${BACKEND_URL}/api/storage?page=1&page_size=100`);
	});

	it('pages through every page and returns all locations, not just the first', async () => {
		const page1 = Array.from({ length: 100 }, (_, i) => loc(i + 1));
		const page2 = Array.from({ length: 100 }, (_, i) => loc(i + 101));
		const page3 = [loc(201), loc(202)];

		const fetchMock = vi.fn().mockImplementation((url: string) => {
			if (url.includes('page=1')) {
				return Promise.resolve(jsonResponse({ data: page1, total_pages: 3, page: 1 }));
			}
			if (url.includes('page=2')) {
				return Promise.resolve(jsonResponse({ data: page2, total_pages: 3, page: 2 }));
			}
			return Promise.resolve(jsonResponse({ data: page3, total_pages: 3, page: 3 }));
		});

		const result = await loadStorageLocations(asFetch(fetchMock));

		expect(fetchMock).toHaveBeenCalledTimes(3);
		expect(result).toHaveLength(202);
		expect(result.map((l) => l.id)).toEqual(Array.from({ length: 202 }, (_, i) => i + 1));
	});

	it('returns an empty array when the first request is not ok', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 500 }));

		const result = await loadStorageLocations(asFetch(fetchMock));

		expect(result).toEqual([]);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it('tolerates a failed later page and returns the pages it could fetch', async () => {
		const page1 = Array.from({ length: 100 }, (_, i) => loc(i + 1));
		const fetchMock = vi.fn().mockImplementation((url: string) => {
			if (url.includes('page=1')) {
				return Promise.resolve(jsonResponse({ data: page1, total_pages: 2, page: 1 }));
			}
			return Promise.resolve(new Response(null, { status: 500 }));
		});

		const result = await loadStorageLocations(asFetch(fetchMock));

		expect(result).toHaveLength(100);
	});

	it('returns an empty array when fetch throws', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('network down'));

		const result = await loadStorageLocations(asFetch(fetchMock));

		expect(result).toEqual([]);
	});
});

describe('loadStorageLocationsWithCounts', () => {
	it('fetches the unbounded with-counts endpoint and returns the bare array', async () => {
		const locations = [
			{ ...loc(1), card_count: 3, item_count: 2, total_value: 10 },
			{ ...loc(2), card_count: 0, item_count: 0, total_value: 0 }
		];
		const fetchMock = vi.fn().mockResolvedValue(jsonResponse(locations));

		const result = await loadStorageLocationsWithCounts(asFetch(fetchMock));

		expect(fetchMock).toHaveBeenCalledTimes(1);
		expect(fetchMock.mock.calls[0][0]).toBe(`${BACKEND_URL}/api/storage/with-counts`);
		expect(result).toHaveLength(2);
	});

	it('returns an empty array when the request is not ok', async () => {
		const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 500 }));

		const result = await loadStorageLocationsWithCounts(asFetch(fetchMock));

		expect(result).toEqual([]);
	});

	it('returns an empty array when fetch throws', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('network down'));

		const result = await loadStorageLocationsWithCounts(asFetch(fetchMock));

		expect(result).toEqual([]);
	});
});
