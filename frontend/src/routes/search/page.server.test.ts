import { describe, it, expect, vi, type Mock } from 'vitest';
import { load } from './+page.server';

const BACKEND_URL = 'http://localhost:3000';

function jsonResponse(body: unknown, status = 200, statusText = ''): Response {
	return new Response(JSON.stringify(body), {
		status,
		statusText,
		headers: { 'content-type': 'application/json' }
	});
}

const searchBody = { data: [{ id: 'card-1' }], page: 1, total_cards: 1, has_more: false };

/**
 * Route fetches by backend path. Storage and settings always succeed; the
 * search response is supplied per test.
 */
function backend(search: () => Promise<Response>): Mock {
	return vi.fn((input: string) => {
		const url = new URL(input);
		if (url.pathname === '/api/search') return search();
		if (url.pathname.startsWith('/api/settings/'))
			return Promise.resolve(jsonResponse({ value: 'ja' }));
		return Promise.resolve(jsonResponse({ data: [], total_pages: 1, page: 1 }));
	});
}

async function runLoad(fetchMock: Mock, query: string) {
	const event = {
		fetch: fetchMock as unknown as typeof globalThis.fetch,
		url: new URL(`http://localhost:5173/search${query}`)
	};
	// load only reads fetch and url; the rest of the event is unused.
	return (await load(event as unknown as Parameters<typeof load>[0])) as Exclude<
		Awaited<ReturnType<typeof load>>,
		void
	>;
}

function searchCalls(fetchMock: Mock): URL[] {
	return fetchMock.mock.calls
		.map(([input]) => new URL(input as string))
		.filter((u) => u.pathname === '/api/search');
}

describe('/search load', () => {
	it('does not search when there is no query', async () => {
		const fetchMock = backend(() => Promise.resolve(jsonResponse(searchBody)));
		const result = await runLoad(fetchMock, '');

		expect(result.search).toBeNull();
		expect(result.lang).toBeNull();
		expect(result.defaultLanguage).toBe('ja');
		expect(searchCalls(fetchMock)).toHaveLength(0);
	});

	it('does not search for a whitespace-only query', async () => {
		const fetchMock = backend(() => Promise.resolve(jsonResponse(searchBody)));
		const result = await runLoad(fetchMock, '?q=%20%20');

		expect(result.search).toBeNull();
		expect(searchCalls(fetchMock)).toHaveLength(0);
	});

	it('searches the backend with q and lang from the URL', async () => {
		const fetchMock = backend(() => Promise.resolve(jsonResponse(searchBody)));
		const result = await runLoad(fetchMock, '?q=lightning+bolt&lang=de');

		const [call] = searchCalls(fetchMock);
		expect(call.origin).toBe(BACKEND_URL);
		expect(call.searchParams.get('q')).toBe('lightning bolt');
		expect(call.searchParams.get('lang')).toBe('de');
		expect(result.lang).toBe('de');
		expect(result.search).toEqual({ query: 'lightning bolt', data: searchBody });
	});

	it('omits a blank lang', async () => {
		const fetchMock = backend(() => Promise.resolve(jsonResponse(searchBody)));
		const result = await runLoad(fetchMock, '?q=bolt&lang=');

		expect(searchCalls(fetchMock)[0].searchParams.has('lang')).toBe(false);
		expect(result.lang).toBeNull();
	});

	it('returns an error result when the backend responds with a failure', async () => {
		const fetchMock = backend(() => Promise.resolve(jsonResponse({}, 502, 'Bad Gateway')));
		const result = await runLoad(fetchMock, '?q=bolt');

		expect(result.search).toEqual({ query: 'bolt', error: 'Search failed: Bad Gateway' });
	});

	it('returns an error result when the request throws', async () => {
		const fetchMock = backend(() => Promise.reject(new Error('connection refused')));
		const result = await runLoad(fetchMock, '?q=bolt');

		expect(result.search).toEqual({ query: 'bolt', error: 'connection refused' });
	});
});
