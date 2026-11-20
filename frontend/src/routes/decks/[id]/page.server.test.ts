import { describe, expect, it, vi } from 'vitest';
import { load } from './+page.server';

const BACKEND_URL = 'http://localhost:3000';

const deck = { id: 7, name: 'Mono Blue', description: '' };
const items = {
	deck_id: 7,
	command: [],
	main: [],
	side: [],
	maybe: [],
	aggregate_shortfall: 0,
	aggregate_over_committed: 0
};
const legality = [{ format: 'modern', name: 'Modern', legal: true, reasons: [] }];

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

// Answers by backend path; an Error rejects the fetch.
function backend(routes: Record<string, Response | Error>) {
	return vi.fn(async (url: string) => {
		const route = routes[url.replace(BACKEND_URL, '')];
		if (route instanceof Error) throw route;
		return route;
	});
}

async function runLoad(fetchMock: ReturnType<typeof backend>) {
	const event = {
		params: { id: '7' },
		fetch: fetchMock as unknown as typeof globalThis.fetch
	} as unknown as Parameters<typeof load>[0];
	return (await load(event)) as Record<string, unknown>;
}

describe('deck detail load', () => {
	it('loads format legality alongside the deck and its items', async () => {
		const result = await runLoad(
			backend({
				'/api/decks/7': json(deck),
				'/api/decks/7/items': json(items),
				'/api/decks/7/legality': json(legality)
			})
		);

		expect(result).toEqual({ deck, items, legality });
	});

	it('still loads the page when the legality check fails', async () => {
		const result = await runLoad(
			backend({
				'/api/decks/7': json(deck),
				'/api/decks/7/items': json(items),
				'/api/decks/7/legality': json({ error: 'boom' }, 500)
			})
		);

		expect(result).toEqual({ deck, items, legality: null });
	});

	it('still loads the page when the legality request rejects', async () => {
		const result = await runLoad(
			backend({
				'/api/decks/7': json(deck),
				'/api/decks/7/items': json(items),
				'/api/decks/7/legality': new Error('connection refused')
			})
		);

		expect(result).toEqual({ deck, items, legality: null });
	});
});
