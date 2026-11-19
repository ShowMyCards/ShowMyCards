import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { EnrichedDeckItem } from '#lib';
import DeckView from './DeckView.svelte';

// Kit's own runtime (e.g. $app/forms) imports afterNavigate from this module, so it must be stubbed too.
vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));

function makeItem(overrides: Partial<EnrichedDeckItem> = {}): EnrichedDeckItem {
	return {
		id: 1,
		created_at: '',
		updated_at: '',
		deck_id: 1,
		oracle_id: 'oracle-1',
		scryfall_id: '',
		treatment: '',
		zone: 'main',
		desired_quantity: 1,
		name: 'Sol Ring',
		set_code: 'cmm',
		collector_number: '703',
		finishes: ['nonfoil', 'foil'],
		image_uri: 'data:image/gif;base64,R0lGODlhAQABAAAAACw=',
		printing_id: 'printing-1',
		owned: 1,
		under_owned: false,
		over_committed: false,
		...overrides
	};
}

const items = [
	makeItem(),
	makeItem({
		id: 2,
		oracle_id: 'oracle-2',
		name: 'Ocelot Pride',
		scryfall_id: 'printing-2',
		printing_id: 'printing-2',
		set_code: 'mh3',
		collector_number: '38',
		treatment: 'foil'
	})
];

const props = {
	items,
	group: 'none' as const,
	sort: 'name' as const,
	getRollup: () => ({ deckWants: 1, spansZones: false })
};

describe('DeckView', () => {
	it('shows the foil overlay only for items with a foil finish', async () => {
		const screen = await render(DeckView, { ...props, view: 'grid' });

		await expect.element(page.getByRole('img', { name: 'Ocelot Pride' })).toBeInTheDocument();
		// "Any finish" ('') must not be mistaken for foil.
		expect(screen.container.querySelectorAll('.foil-overlay')).toHaveLength(1);
	});

	it('marks foil and pinned items in the text view', async () => {
		await render(DeckView, { ...props, view: 'text' });

		const pinned = page.getByLabelText('Pinned to MH3 #38', { exact: false });
		await expect.element(pinned).toBeInTheDocument();
		expect(page.getByLabelText(/^Pinned to/).elements()).toHaveLength(1);
		expect(page.getByLabelText('Foil', { exact: true }).elements()).toHaveLength(1);
	});
});
