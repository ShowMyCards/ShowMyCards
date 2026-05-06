import { BACKEND_URL } from '$lib';
import { resolveBackendPath } from '$lib/api/path-resolver';
import type { RequestHandler } from './$types';

/**
 * Catch-all proxy for `/api/*` requests from the browser.
 *
 * The browser cannot reach BACKEND_URL directly (it points at a host that's
 * only routable from inside the SvelteKit server — e.g. a Docker network
 * address). This handler strips the `/api` prefix and forwards every method
 * to the corresponding backend path, streaming the body and preserving the
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

const proxy: RequestHandler = async ({ params, request, url, fetch }) => {
	const path = params.path ?? '';
	const target = `${BACKEND_URL}${resolveBackendPath(path)}${url.search}`;

	const init: RequestInit = { method: request.method };
	const forwardedRequestHeaders = new Headers();
	const contentType = request.headers.get('content-type');
	if (contentType) forwardedRequestHeaders.set('content-type', contentType);
	const accept = request.headers.get('accept');
	if (accept) forwardedRequestHeaders.set('accept', accept);
	init.headers = forwardedRequestHeaders;

	if (request.method !== 'GET' && request.method !== 'HEAD') {
		init.body = await request.arrayBuffer();
	}

	const upstream = await fetch(target, init);

	const headers = new Headers();
	for (const name of RESPONSE_HEADERS_TO_FORWARD) {
		const value = upstream.headers.get(name);
		if (value) headers.set(name, value);
	}

	return new Response(upstream.body, {
		status: upstream.status,
		statusText: upstream.statusText,
		headers
	});
};

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const DELETE = proxy;
export const PATCH = proxy;
