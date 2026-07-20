import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { EnrichedDeckItem } from '$lib';
import DeckItemRow from './DeckItemRow.svelte';

function makeItem(overrides: Partial<EnrichedDeckItem> = {}): EnrichedDeckItem {
	return {
		id: 1,
		created_at: '2026-01-01T00:00:00Z',
		updated_at: '2026-01-01T00:00:00Z',
		deck_id: 1,
		oracle_id: 'oracle-1',
		scryfall_id: '',
		treatment: '',
		zone: 'main',
		desired_quantity: 1,
		name: "Ashnod's Altar",
		set_name: 'Brothers War Retro',
		collector_number: '67',
		finishes: ['nonfoil'],
		owned: 1,
		under_owned: false,
		over_committed: false,
		...overrides
	};
}

describe('DeckItemRow', () => {
	it('renders the card name', async () => {
		render(DeckItemRow, { item: makeItem(), deckWants: 1 });
		await expect.element(page.getByText("Ashnod's Altar")).toBeInTheDocument();
	});

	it('shows the availability readout for a demand-zone item', async () => {
		render(DeckItemRow, { item: makeItem({ owned: 2 }), deckWants: 3 });
		await expect
			.element(page.getByTestId('availability'))
			.toHaveTextContent('own 2 / deck wants 3');
	});

	it('shows the under-owned badge when under_owned is set', async () => {
		render(DeckItemRow, { item: makeItem({ under_owned: true, owned: 1 }), deckWants: 4 });
		await expect.element(page.getByTestId('under-owned-badge')).toBeInTheDocument();
	});

	it('shows the over-committed badge (not under-owned) when only over_committed is set', async () => {
		render(DeckItemRow, {
			item: makeItem({ over_committed: true, under_owned: false }),
			deckWants: 1
		});
		await expect.element(page.getByTestId('over-committed-badge')).toBeInTheDocument();
		expect(page.getByTestId('under-owned-badge').query()).toBeNull();
	});

	it('notes "across zones" when the card is split across demand zones', async () => {
		render(DeckItemRow, {
			item: makeItem({ zone: 'main', desired_quantity: 3, under_owned: true }),
			deckWants: 4,
			spansZones: true
		});
		await expect.element(page.getByTestId('availability')).toHaveTextContent('across zones');
	});

	it('does not repeat the badge on a secondary split-zone row', async () => {
		render(DeckItemRow, {
			item: makeItem({ id: 2, zone: 'side', under_owned: true }),
			deckWants: 4,
			spansZones: true,
			showAvailability: false
		});
		// Secondary row cross-references the rollup instead of showing another badge.
		await expect.element(page.getByTestId('rollup-note')).toHaveTextContent('counts toward 4');
		expect(page.getByTestId('under-owned-badge').query()).toBeNull();
		expect(page.getByTestId('availability').query()).toBeNull();
	});

	it('shows only the owned count for maybe-board items (no demand signal)', async () => {
		render(DeckItemRow, { item: makeItem({ zone: 'maybe', owned: 5 }), deckWants: 0 });
		await expect.element(page.getByText('own 5')).toBeInTheDocument();
		expect(page.getByTestId('availability').query()).toBeNull();
		expect(page.getByTestId('under-owned-badge').query()).toBeNull();
	});
});
