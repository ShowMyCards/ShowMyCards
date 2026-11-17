import { page, userEvent } from 'vitest/browser';
import { afterEach, describe, expect, it, vi, type MockInstance } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { EnrichedDeckItem } from '#lib';
import DeckCardActions from './DeckCardActions.svelte';

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
		zone: 'side',
		desired_quantity: 1,
		name: 'Ocelot Pride',
		set_code: 'mh3',
		collector_number: '38',
		finishes: ['nonfoil'],
		image_uri: '',
		printing_id: 'printing-1',
		owned: 1,
		under_owned: false,
		over_committed: false,
		...overrides
	};
}

let fetchSpy: MockInstance<typeof fetch>;

/** Answer every form action with a successful SvelteKit action result. */
function mockActions() {
	fetchSpy = vi
		.spyOn(globalThis, 'fetch')
		.mockImplementation(
			async () => new Response(JSON.stringify({ type: 'success', status: 200 }), { status: 200 })
		);
}

/** The form fields posted to the given action, if it was called. */
function posted(action: string) {
	const call = fetchSpy.mock.calls.find(([input]) => String(input).endsWith(`?/${action}`));
	if (!call) return undefined;
	return Object.fromEntries(call[1]?.body as FormData);
}

function menus() {
	return Array.from(document.querySelectorAll('details[name="deck-card-actions"]'));
}

describe('DeckCardActions', () => {
	afterEach(() => vi.restoreAllMocks());

	it('moves a single copy as soon as a zone is picked', async () => {
		mockActions();
		await render(DeckCardActions, { item: makeItem(), deckWants: 1 });

		await page.getByLabelText('Card actions for Ocelot Pride').click();
		await page.getByLabelText('Zone').selectOptions('main');

		await vi.waitFor(() =>
			expect(posted('moveItem')).toEqual({ item_id: '1', zone: 'main', quantity: '1' })
		);
	});

	it('asks how many to move when there is more than one copy', async () => {
		mockActions();
		await render(DeckCardActions, { item: makeItem({ desired_quantity: 3 }), deckWants: 3 });

		await page.getByLabelText('Card actions for Ocelot Pride').click();
		await page.getByLabelText('Zone').selectOptions('main');
		expect(posted('moveItem')).toBeUndefined();

		await expect.element(page.getByLabelText('Quantity to move')).toHaveValue(3);
		await page.getByLabelText('Quantity to move').fill('2');
		await page.getByRole('button', { name: 'Move', exact: true }).click();

		await vi.waitFor(() =>
			expect(posted('moveItem')).toEqual({ item_id: '1', zone: 'main', quantity: '2' })
		);
	});

	it('cancelling a pending move restores the current zone', async () => {
		mockActions();
		await render(DeckCardActions, { item: makeItem({ desired_quantity: 2 }), deckWants: 2 });

		await page.getByLabelText('Card actions for Ocelot Pride').click();
		await page.getByLabelText('Zone').selectOptions('main');
		await page.getByRole('button', { name: 'Cancel' }).click();

		await expect.element(page.getByLabelText('Zone')).toHaveValue('side');
		await expect.element(page.getByLabelText('Quantity to move')).not.toBeInTheDocument();
		expect(posted('moveItem')).toBeUndefined();
	});

	it('keeps only one row menu open at a time', async () => {
		await render(DeckCardActions, { item: makeItem(), deckWants: 1 });
		await render(DeckCardActions, { item: makeItem({ id: 2, name: 'Sol Ring' }), deckWants: 1 });

		await page.getByLabelText('Card actions for Ocelot Pride').click();
		await page.getByLabelText('Card actions for Sol Ring').click();

		await vi.waitFor(() =>
			expect(menus().map((m) => (m as HTMLDetailsElement).open)).toEqual([false, true])
		);
	});

	it('closes on Escape and on a click outside the menu', async () => {
		await render(DeckCardActions, { item: makeItem(), deckWants: 1 });
		const menu = menus()[0] as HTMLDetailsElement;
		const toggle = page.getByLabelText('Card actions for Ocelot Pride');

		await toggle.click();
		expect(menu.open).toBe(true);
		await userEvent.keyboard('{Escape}');
		await vi.waitFor(() => expect(menu.open).toBe(false));

		await toggle.click();
		expect(menu.open).toBe(true);
		document.body.click();
		await vi.waitFor(() => expect(menu.open).toBe(false));
	});
});
