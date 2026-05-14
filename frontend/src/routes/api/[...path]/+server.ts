import { BACKEND_URL } from '$lib';
import { resolveBackendPath } from '$lib/api/path-resolver';
import type { RequestHandler } from './$types';

/**
 * Catch-all proxy for `/api/*` requests from the browser.
 *
 * The browser cannot reach BACKEND_URL directly (it points at a host that's
 * only routable from inside the SvelteKit server — e.g. a Docker network
 * address). This handler strips the `/api` prefix and forwards every method
 * to the corresponding backend path, buffering the body and preserving the
 * headers that callers care about (downloads, caching, content-type).
 *
 * Server-side code (+page.server.ts, etc.) should keep calling BACKEND_URL
 * directly; only client-side fetches need this proxy.
 */
const RESPONSE_HEADERS_TO_FORWARD = [
	'content-type',
	'content-disposition',
	'cache-control',
	'etag',
	'last-modified'
];

const REQUEST_HEADERS_TO_FORWARD = [
	'content-type',
	'accept',
	'accept-encoding',
	'if-none-match',
	'if-modified-since'
];

const proxy: RequestHandler = async ({ params, request, url, fetch }) => {
	const path = params.path ?? '';
	const target = `${BACKEND_URL}${resolveBackendPath(path)}${url.search}`;

	const requestHeaders = new Headers();
	for (const name of REQUEST_HEADERS_TO_FORWARD) {
		const value = request.headers.get(name);
		if (value) requestHeaders.set(name, value);
	}

	const init: RequestInit = { method: request.method, headers: requestHeaders };
	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.arrayBuffer();
	}

	const upstream = await fetch(target, init);

	const responseHeaders = new Headers();
	for (const name of RESPONSE_HEADERS_TO_FORWARD) {
		const value = upstream.headers.get(name);
		if (value) responseHeaders.set(name, value);
	}

	return new Response(upstream.body, {
		status: upstream.status,
		statusText: upstream.statusText,
		headers: responseHeaders
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
