import { describe, it, expect, vi, type Mock } from 'vitest';
import { GET, POST, PUT, DELETE, PATCH } from './+server';

const BACKEND_URL = 'http://localhost:3000';

const handlers = { GET, POST, PUT, DELETE, PATCH };
type Method = keyof typeof handlers;

interface EventOpts {
	path?: string;
	method?: Method;
	search?: string;
	headers?: Record<string, string>;
	body?: BodyInit;
}

function makeEvent(opts: EventOpts, fetchMock: Mock) {
	const { path, method = 'GET', search = '', headers, body } = opts;
	const requestUrl = `http://localhost/api/${path ?? ''}${search}`;
	const requestInit: RequestInit = { method, headers };
	if (body !== undefined) requestInit.body = body;
	return {
		params: { path },
		request: new Request(requestUrl, requestInit),
		url: new URL(requestUrl),
		fetch: fetchMock as unknown as typeof globalThis.fetch
	};
}

function mockFetch(response: Response = new Response(null, { status: 200 })): Mock {
	return vi.fn().mockResolvedValue(response);
}

async function call(method: Method, event: ReturnType<typeof makeEvent>) {
	// The event we build only includes the fields the proxy reads; the full
	// RequestEvent type has many more, so we cast.
	return handlers[method](event as Parameters<(typeof handlers)[Method]>[0]);
}

function fetchInit(fetchMock: Mock): RequestInit {
	return fetchMock.mock.calls[0][1] as RequestInit;
}

function fetchUrl(fetchMock: Mock): string {
	return fetchMock.mock.calls[0][0] as string;
}

describe('catch-all proxy: upstream URL', () => {
	it('prefixes /api/ for API-segment paths', async () => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(fetchUrl(fetchMock)).toBe(`${BACKEND_URL}/api/jobs`);
	});

	it('leaves root-hosted paths at the root', async () => {
		const fetchMock = mockFetch();
		await call('POST', makeEvent({ path: 'inventory/batch/move', method: 'POST' }, fetchMock));
		expect(fetchUrl(fetchMock)).toBe(`${BACKEND_URL}/inventory/batch/move`);
	});

	it('appends the query string verbatim', async () => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'search', search: '?q=foo&page=2' }, fetchMock));
		expect(fetchUrl(fetchMock)).toBe(`${BACKEND_URL}/search?q=foo&page=2`);
	});

	it('falls back to "/" when params.path is undefined', async () => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: undefined }, fetchMock));
		expect(fetchUrl(fetchMock)).toBe(`${BACKEND_URL}/`);
	});
});

describe('catch-all proxy: method forwarding', () => {
	it.each(['GET', 'POST', 'PUT', 'DELETE', 'PATCH'] as const)(
		'forwards %s to the upstream',
		async (method) => {
			const fetchMock = mockFetch();
			await call(method, makeEvent({ path: 'jobs', method }, fetchMock));
			expect(fetchInit(fetchMock).method).toBe(method);
		}
	);
});

describe('catch-all proxy: request headers', () => {
	it.each([
		['content-type', 'application/json'],
		['accept', 'application/json']
	])('forwards %s', async (header, value) => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'jobs', headers: { [header]: value } }, fetchMock));
		const sent = new Headers(fetchInit(fetchMock).headers);
		expect(sent.get(header)).toBe(value);
	});

	it.each(['authorization', 'cookie', 'x-internal'])('drops %s', async (header) => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'jobs', headers: { [header]: 'leak' } }, fetchMock));
		const sent = new Headers(fetchInit(fetchMock).headers);
		expect(sent.has(header)).toBe(false);
	});

	it('does not invent allow-listed headers when caller omits them', async () => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		const sent = new Headers(fetchInit(fetchMock).headers);
		expect(sent.has('content-type')).toBe(false);
		expect(sent.has('accept')).toBe(false);
	});
});

describe('catch-all proxy: request body', () => {
	it('does not send a body for GET', async () => {
		const fetchMock = mockFetch();
		await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(fetchInit(fetchMock).body).toBeUndefined();
	});

	it('forwards the body bytes for POST', async () => {
		const fetchMock = mockFetch();
		const payload = JSON.stringify({ hello: 'world' });
		await call(
			'POST',
			makeEvent(
				{
					path: 'jobs',
					method: 'POST',
					headers: { 'content-type': 'application/json' },
					body: payload
				},
				fetchMock
			)
		);
		const body = fetchInit(fetchMock).body as ArrayBuffer;
		expect(new TextDecoder().decode(body)).toBe(payload);
	});

	it('forwards the body bytes for DELETE', async () => {
		const fetchMock = mockFetch();
		const payload = JSON.stringify({ ids: [1, 2, 3] });
		await call(
			'DELETE',
			makeEvent(
				{
					path: 'inventory/batch',
					method: 'DELETE',
					headers: { 'content-type': 'application/json' },
					body: payload
				},
				fetchMock
			)
		);
		const body = fetchInit(fetchMock).body as ArrayBuffer;
		expect(new TextDecoder().decode(body)).toBe(payload);
	});
});

describe('catch-all proxy: response status', () => {
	it.each([
		[201, 'Created'],
		[404, 'Not Found'],
		[500, 'Internal Server Error']
	])('passes through status %i / %s', async (status, statusText) => {
		const fetchMock = mockFetch(new Response(null, { status, statusText }));
		const response = await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(response.status).toBe(status);
		expect(response.statusText).toBe(statusText);
	});
});

describe('catch-all proxy: response headers', () => {
	it.each([
		['content-type', 'application/json'],
		['content-disposition', 'attachment; filename="export.json"'],
		['cache-control', 'no-store'],
		['etag', '"abc123"'],
		['last-modified', 'Wed, 21 Oct 2026 07:28:00 GMT']
	])('forwards %s', async (header, value) => {
		const fetchMock = mockFetch(new Response(null, { headers: { [header]: value } }));
		const response = await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(response.headers.get(header)).toBe(value);
	});

	it.each(['set-cookie', 'x-internal-debug', 'server'])('drops %s', async (header) => {
		const fetchMock = mockFetch(new Response(null, { headers: { [header]: 'leak' } }));
		const response = await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(response.headers.has(header)).toBe(false);
	});

	it('does not invent allow-listed headers when upstream omits them', async () => {
		const fetchMock = mockFetch(new Response(null, { status: 204 }));
		const response = await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(response.headers.has('content-type')).toBe(false);
		expect(response.headers.has('etag')).toBe(false);
	});
});

describe('catch-all proxy: response body', () => {
	it('returns the upstream body unchanged', async () => {
		const payload = JSON.stringify({ ok: true, items: [1, 2, 3] });
		const fetchMock = mockFetch(
			new Response(payload, { headers: { 'content-type': 'application/json' } })
		);
		const response = await call('GET', makeEvent({ path: 'jobs' }, fetchMock));
		expect(await response.text()).toBe(payload);
	});
});
