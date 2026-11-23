import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { EnhancedCardResult } from '#lib';
import DeckAddResultCard from './DeckAddResultCard.svelte';

vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));

function makeCard(overrides: Partial<EnhancedCardResult> = {}): EnhancedCardResult {
	return {
		id: 'bolt-de',
		oracle_id: 'oracle-bolt',
		name: 'Lightning Bolt',
		printed_name: 'Blitzschlag',
		set_code: 'm10',
		set_name: 'Magic 2010',
		collector_number: '146',
		language: 'de',
		color_identity: ['R'],
		finishes: ['nonfoil'],
		prices: {},
		image_uri: 'data:image/gif;base64,R0lGODlhAQABAAAAACw=',
		inventory: {
			this_printing: [],
			other_printings: [],
			total_quantity: 0
		},
		...overrides
	};
}

describe('DeckAddResultCard', () => {
	it('displays printed_name and language badge for non-English cards', async () => {
		await render(DeckAddResultCard, { card: makeCard() });

		await expect.element(page.getByText('Blitzschlag')).toBeInTheDocument();
		await expect.element(page.getByRole('img', { name: 'Blitzschlag' })).toBeInTheDocument();
		await expect.element(page.getByText('de', { exact: true })).toBeInTheDocument();
	});

	it('falls back to English name when printed_name is absent', async () => {
		await render(DeckAddResultCard, {
			card: makeCard({ id: 'bolt-en', printed_name: undefined, language: 'en' })
		});

		await expect.element(page.getByText('Lightning Bolt')).toBeInTheDocument();
		await expect.element(page.getByText('en', { exact: true })).toBeInTheDocument();
	});
});
