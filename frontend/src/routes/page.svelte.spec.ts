import { page } from 'vitest/browser';
import { afterEach, beforeEach, describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import Page from './+page.svelte';
import type { PageData } from './$types';
import { currency } from '$lib/stores/currency.svelte';

// +page.svelte expects the shape returned by its server load. Rendering the
// component in isolation means supplying it directly.
const data: PageData = {
	stats: {
		total_inventory_cards: 0,
		total_wishlist_cards: 0,
		total_collection_value: 0,
		total_collected_from_lists: 0,
		total_remaining_lists_value: 0,
		total_collection_value_eur: 0,
		total_collected_from_lists_eur: 0,
		total_remaining_lists_value_eur: 0,
		total_storage_locations: 0,
		total_lists: 0,
		unassigned_cards: 0
	}
};

describe('/+page.svelte', () => {
	beforeEach(() => {
		currency.set('usd');
	});
	afterEach(() => {
		currency.set('usd');
	});

	it('uses the selected currency for all dashboard values', async () => {
		currency.set('eur');
		render(Page, { data });

		expect(page.getByText('€0.00', { exact: true }).elements()).toHaveLength(3);
		expect(page.getByText('$0.00', { exact: true }).elements()).toHaveLength(0);
	});

	it.each(['usd', 'eur'] as const)(
		'displays %s totals and updates all values when currency changes',
		async (selectedCurrency) => {
			currency.set(selectedCurrency);
			render(Page, {
				data: {
					stats: {
						...data.stats,
						total_collection_value: 120,
						total_collected_from_lists: 60,
						total_remaining_lists_value: 180,
						total_collection_value_eur: 36,
						total_collected_from_lists_eur: 18,
						total_remaining_lists_value_eur: 54
					}
				}
			});
			const expectedValues = {
				usd: ['$120.00', '$60.00', '$180.00'],
				eur: ['€36.00', '€18.00', '€54.00']
			};
			for (const value of expectedValues[selectedCurrency]) {
				await expect.element(page.getByText(value, { exact: true })).toBeInTheDocument();
			}

			const nextCurrency = selectedCurrency === 'usd' ? 'eur' : 'usd';
			currency.set(nextCurrency);
			for (const value of expectedValues[nextCurrency]) {
				await expect.element(page.getByText(value, { exact: true })).toBeInTheDocument();
			}
			for (const value of expectedValues[selectedCurrency]) {
				await expect.element(page.getByText(value, { exact: true })).not.toBeInTheDocument();
			}
		}
	);

	it('should render h1', async () => {
		render(Page, { data });

		const heading = page.getByRole('heading', { level: 1 });
		await expect.element(heading).toBeInTheDocument();
	});
});
