import { describe, expect, it, vi, type Mock } from 'vitest';
import { POST } from './+server';

const BACKEND_URL = 'http://localhost:3000';

const item = {
	oracle_id: 'sol',
	scryfall_id: '',
	treatment: '',
	desired_quantity: 2,
	zone: 'main'
};

function makeEvent(body: unknown, fetchMock: Mock) {
	const request = new Request('http://localhost/decks/7/items', {
		method: 'POST',
		headers: { 'content-type': 'application/json' },
		body: typeof body === 'string' ? body : JSON.stringify(body)
	});
	// Only the fields the handler reads; the full RequestEvent has many more.
	return {
		params: { id: '7' },
		request,
		fetch: fetchMock as unknown as typeof globalThis.fetch
	} as unknown as Parameters<typeof POST>[0];
}

const emptyDeck = () =>
	new Response(JSON.stringify({ command: [], main: [], side: [], maybe: [] }), { status: 200 });

describe('POST /decks/[id]/items', () => {
	it('creates new items through the batch endpoint', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(emptyDeck())
			.mockResolvedValueOnce(new Response('[]', { status: 201 }));

		const res = await POST(makeEvent({ items: [item] }, fetchMock));

		expect(res.status).toBe(200);
		expect(await res.json()).toEqual({ success: true });
		const [url, init] = fetchMock.mock.calls[1];
		expect(url).toBe(`${BACKEND_URL}/api/decks/7/items/batch`);
		expect(JSON.parse((init as RequestInit).body as string)).toEqual({ items: [item] });
	});

	it('increments an item that is already in the deck', async () => {
		const existing = { ...item, id: 42, desired_quantity: 1 };
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(new Response(JSON.stringify({ main: [existing] }), { status: 200 }))
			.mockResolvedValueOnce(new Response('{}', { status: 200 }));

		const res = await POST(makeEvent({ items: [item] }, fetchMock));

		expect(res.status).toBe(200);
		const [url, init] = fetchMock.mock.calls[1];
		expect(url).toBe(`${BACKEND_URL}/api/decks/7/items/42`);
		expect((init as RequestInit).method).toBe('PUT');
		expect(JSON.parse((init as RequestInit).body as string)).toEqual({ desired_quantity: 3 });
	});

	it.each([[{}], [{ items: [] }], ['not json']])('rejects %j with 400', async (body) => {
		const fetchMock = vi.fn();
		const res = await POST(makeEvent(body, fetchMock));
		expect(res.status).toBe(400);
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('passes through the upstream failure status and message', async () => {
		const fetchMock = vi
			.fn()
			.mockResolvedValueOnce(emptyDeck())
			.mockResolvedValueOnce(
				new Response(JSON.stringify({ error: 'deck not found' }), { status: 404 })
			);

		const res = await POST(makeEvent({ items: [item] }, fetchMock));

		expect(res.status).toBe(404);
		expect(await res.json()).toEqual({ error: 'deck not found' });
	});
});
