import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { inventoryCard, inventoryItem } from '#lib/test/fixtures/inventory.ts';
import type { Inventory, InventoryCard } from '#lib';
import Page from './+page.svelte';
import type { PageData } from './$types';

// #lib re-exports and Kit's runtime import refreshAll and afterNavigate from this module.
vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));

function makeData(
	items: Inventory[],
	options: { card?: Partial<InventoryCard>; otherPrintings?: InventoryCard[] } = {}
): PageData {
	return {
		card: inventoryCard('card-1', items, options.card),
		otherPrintings: options.otherPrintings ?? [],
		deckUsage: []
	} as PageData;
}

describe('/cards/[id]/+page.svelte', () => {
	it('renders clickable links for owned non-English printings with a language badge', async () => {
		const japaneseOwned = inventoryCard(
			'ja-1',
			[inventoryItem(10, { scryfall_id: 'ja-1', quantity: 2 })],
			{
				set_name: 'Born of the Gods',
				language: 'ja',
				printed_name: 'オレスコスの王、ブリマーズ'
			}
		);
		const englishUnowned = inventoryCard('en-2', [], {
			set_name: 'Masters 25',
			language: 'en'
		});

		await render(Page, {
			data: makeData([], { otherPrintings: [japaneseOwned, englishUnowned] })
		});

		const jaLink = page.getByRole('link', { name: /Born of the Gods/ });
		await expect.element(jaLink).toHaveAttribute('href', '/cards/ja-1');
		await expect.element(jaLink.getByText('2 owned')).toBeInTheDocument();
		await expect.element(jaLink.getByText('ja')).toBeInTheDocument();

		const enLink = page.getByRole('link', { name: /Masters 25/ });
		await expect.element(enLink).toHaveAttribute('href', '/cards/en-2');
		await expect.element(enLink.getByText('en')).not.toBeInTheDocument();
	});
});
