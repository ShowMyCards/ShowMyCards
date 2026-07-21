import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { EnhancedCardResult } from '$lib';
import CardResultCard from './CardResultCard.svelte';

function makeCard(overrides: Partial<EnhancedCardResult> = {}): EnhancedCardResult {
	return {
		id: 'card-1',
		oracle_id: 'oracle-1',
		name: 'Sol Ring',
		set_code: 'cmm',
		set_name: 'Commander Masters',
		collector_number: '1',
		language: 'en',
		color_identity: [],
		finishes: ['nonfoil'],
		prices: {},
		inventory: { this_printing: [], other_printings: [], total_quantity: 4 },
		...overrides
	} as EnhancedCardResult;
}

describe('CardResultCard deck availability indicator', () => {
	it('shows free and decked counts when a deck summary is provided', async () => {
		render(CardResultCard, { card: makeCard(), deck: { owned: 4, decked: 1, free: 3 } });
		await expect.element(page.getByText('3 free for decks')).toBeInTheDocument();
		await expect.element(page.getByText('1 decked')).toBeInTheDocument();
	});

	it('omits the "decked" note when nothing is committed', async () => {
		render(CardResultCard, { card: makeCard(), deck: { owned: 4, decked: 0, free: 4 } });
		await expect.element(page.getByText('4 free for decks')).toBeInTheDocument();
		expect(page.getByText('decked').query()).toBeNull();
	});

	it('renders no deck indicator when no deck summary is provided', async () => {
		render(CardResultCard, { card: makeCard() });
		expect(page.getByText('free for decks').query()).toBeNull();
	});
});
