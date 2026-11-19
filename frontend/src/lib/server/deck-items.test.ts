import { describe, expect, it, vi } from 'vitest';
import { upsertDeckItems, type DeckItemInput } from './deck-items';

const rabbit: DeckItemInput = {
	oracle_id: 'rabbit',
	scryfall_id: 'tblb-1',
	treatment: '',
	desired_quantity: 1,
	zone: 'main'
};

/** A fetch stub: the deck's current items, then canned replies per method. */
function backend(existing: Record<string, unknown[]>, created: unknown) {
	return vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
		if (!init?.method) return new Response(JSON.stringify(existing), { status: 200 });
		if (init.method === 'POST') return new Response(JSON.stringify(created), { status: 201 });
		return new Response(JSON.stringify({}), { status: 200 });
	}) as unknown as typeof fetch;
}

describe('upsertDeckItems', () => {
	it('reports the zones the backend stored new rows in', async () => {
		// Asked for main; the backend keeps tokens in other.
		const fetch = backend({}, [{ ...rabbit, id: 1, zone: 'other' }]);

		const result = await upsertDeckItems(fetch, 7, [rabbit]);

		expect(result).toEqual({ ok: true, zones: ['other'] });
	});

	it('reports the zone of an existing row it increments', async () => {
		const fetch = backend({ side: [{ ...rabbit, id: 3, zone: 'side', desired_quantity: 2 }] }, []);

		const result = await upsertDeckItems(fetch, 7, [{ ...rabbit, zone: 'side' }]);

		expect(result).toEqual({ ok: true, zones: ['side'] });
	});
});
