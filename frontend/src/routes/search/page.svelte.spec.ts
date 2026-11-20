import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { InventoryCard } from '#lib';
import Page from './+page.svelte';
import type { PageData } from './$types';

// Kit's own runtime (e.g. $app/forms) imports afterNavigate from this module, so it must be stubbed too.
vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));

function makeCard(id: string, name: string): InventoryCard {
	return {
		id,
		oracle_id: `oracle-${id}`,
		name,
		set_code: 'cmm',
		set_name: 'Commander Masters',
		collector_number: '1',
		language: 'en',
		color_identity: [],
		finishes: ['nonfoil'],
		prices: {},
		inventory: { this_printing: [], other_printings: [], total_quantity: 0 },
		deck: { owned: 0, decked: 0, free: 0 }
	} as unknown as InventoryCard;
}

function makeData(overrides: Partial<PageData> = {}): PageData {
	return {
		storageLocations: [],
		defaultLanguage: 'en',
		search: null,
		lang: null,
		...overrides
	} as PageData;
}

describe('/search/+page.svelte', () => {
	it('renders a GET form with q and lang fields so searches live in the URL', async () => {
		const { container } = await render(Page, { data: makeData() });

		const form = container.querySelector('form');
		expect(form?.method).toBe('get');
		expect(new URL(form!.action).pathname).toBe('/search');
		expect(form?.querySelector('input[name="q"]')).not.toBeNull();
		expect(form?.querySelector('select[name="lang"]')).not.toBeNull();
	});

	it('shows no results block without a search', async () => {
		await render(Page, { data: makeData() });
		expect(page.getByText('Results Found').query()).toBeNull();
	});

	it('pre-fills the query and language and renders results from the URL search', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(new Response('[]', { status: 200 }));
		await render(Page, {
			data: makeData({
				lang: 'de',
				search: {
					query: 'sol ring',
					data: {
						data: [makeCard('a', 'Sol Ring'), makeCard('b', 'Sol Talisman')],
						page: 1,
						total_cards: 2,
						has_more: false
					}
				}
			})
		});

		await expect.element(page.getByPlaceholder('Search for cards...')).toHaveValue('sol ring');
		await expect.element(page.getByRole('combobox', { name: 'Search language' })).toHaveValue('de');
		await expect.element(page.getByText('Results Found')).toBeInTheDocument();
		await expect.element(page.getByText('Sol Talisman').first()).toBeInTheDocument();
		vi.restoreAllMocks();
	});

	it('shows no results block when the search failed', async () => {
		await render(Page, {
			data: makeData({ search: { query: 'bolt', error: 'Search failed: Bad Gateway' } })
		});

		await expect.element(page.getByPlaceholder('Search for cards...')).toHaveValue('bolt');
		expect(page.getByText('Results Found').query()).toBeNull();
	});
});
