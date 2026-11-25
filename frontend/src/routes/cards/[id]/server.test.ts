import { describe, expect, it, vi, type Mock } from 'vitest';
import { load } from './+page.server';
import type { PageData } from './$types';

type Route = (url: URL) => Response | Promise<Response>;

const json = (body: unknown, status = 200) => new Response(JSON.stringify(body), { status });

const englishCard = {
	id: 'en-1',
	oracle_id: 'oracle-brimaz',
	name: 'Brimaz, King of Oreskos',
	language: 'en'
};

const japanesePrinting = {
	id: 'ja-1',
	oracle_id: 'oracle-brimaz',
	name: 'Brimaz, King of Oreskos',
	printed_name: 'オレスコスの王、ブリマーズ',
	language: 'ja',
	inventory: {
		this_printing: [{ id: 1, quantity: 2 }],
		other_printings: [],
		total_quantity: 2
	}
};

function backend(routes: { card?: Route; printings?: Route; decks?: Route } = {}): Mock {
	return vi.fn(async (input: RequestInfo | URL) => {
		const url = new URL(String(input));
		if (url.pathname.startsWith('/api/cards/printings/') && routes.printings) {
			return routes.printings(url);
		}
		if (url.pathname.startsWith('/api/cards/') && routes.card) {
			return routes.card(url);
		}
		if (url.pathname.startsWith('/api/decks/for-card/') && routes.decks) {
			return routes.decks(url);
		}
		return new Response('not found', { status: 404 });
	});
}

async function run(id: string, fetchMock: Mock): Promise<PageData> {
	const event = {
		params: { id },
		fetch: fetchMock as unknown as typeof globalThis.fetch
	} as unknown as Parameters<typeof load>[0];
	const data = await load(event);
	if (!data) throw new Error('load returned nothing');
	return data as PageData;
}

function requestTo(fetchMock: Mock, prefix: string): URL | undefined {
	const call = fetchMock.mock.calls.find(([input]) =>
		new URL(String(input)).pathname.startsWith(prefix)
	);
	return call ? new URL(String(call[0])) : undefined;
}

describe('/cards/[id] load', () => {
	it('fetches /api/cards/printings/:oracle_id with card language, excludes current card, and keeps owned non-English printings', async () => {
		const fetchMock = backend({
			card: () => json(englishCard),
			printings: () => json([englishCard, japanesePrinting]),
			decks: () => json([])
		});

		const data = await run('en-1', fetchMock);

		const printingsUrl = requestTo(fetchMock, '/api/cards/printings/');
		expect(printingsUrl?.pathname).toBe('/api/cards/printings/oracle-brimaz');
		expect(printingsUrl?.searchParams.get('lang')).toBe('en');
		expect(data.otherPrintings.map((p) => p.id)).toEqual(['ja-1']);
	});

	it('omits lang query param when card language is empty', async () => {
		const fetchMock = backend({
			card: () => json({ ...englishCard, language: '' }),
			printings: () => json([japanesePrinting])
		});

		await run('en-1', fetchMock);

		const printingsUrl = requestTo(fetchMock, '/api/cards/printings/');
		expect(printingsUrl?.pathname).toBe('/api/cards/printings/oracle-brimaz');
		expect(printingsUrl?.searchParams.has('lang')).toBe(false);
	});

	it('skips printings fetch when card has no oracle_id', async () => {
		const fetchMock = backend({
			card: () => json({ ...englishCard, oracle_id: '' })
		});

		const data = await run('en-1', fetchMock);

		expect(requestTo(fetchMock, '/api/cards/printings/')).toBeUndefined();
		expect(data.otherPrintings).toEqual([]);
	});

	it('degrades to empty otherPrintings when printings endpoint fails or throws', async () => {
		const failingMock = backend({
			card: () => json(englishCard),
			printings: () => new Response('boom', { status: 500 })
		});
		expect((await run('en-1', failingMock)).otherPrintings).toEqual([]);

		const throwingMock = backend({
			card: () => json(englishCard),
			printings: () => {
				throw new TypeError('network error');
			}
		});
		expect((await run('en-1', throwingMock)).otherPrintings).toEqual([]);
	});
});
