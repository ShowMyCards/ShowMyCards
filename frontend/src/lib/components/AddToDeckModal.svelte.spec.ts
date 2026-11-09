import { page } from 'vitest/browser';
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { DeckSummary } from '$lib';
import AddToDeckModal from './AddToDeckModal.svelte';

function deck(id: number, name: string): DeckSummary {
	return {
		id,
		name,
		description: '',
		created_at: '',
		updated_at: '',
		total_items: 0,
		total_cards_demand: 0,
		aggregate_shortfall: 0
	};
}

const sol = { oracle_id: 'sol', scryfall_id: 'cmm-1', name: 'Sol Ring', copies: 1 };

let fetchSpy: MockInstance<typeof fetch>;

/** Route the deck list (via the /api proxy) and the add endpoint to canned responses. */
function mockBackend(decks: DeckSummary[]) {
	fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
		const url = String(input);
		if (url.endsWith('/api/decks')) {
			return new Response(JSON.stringify(decks), { status: 200 });
		}
		if (/\/decks\/\d+\/items$/.test(url)) {
			return new Response(JSON.stringify({ success: true }), { status: 200 });
		}
		return new Response('not found', { status: 404 });
	});
}

function postedItems() {
	const call = fetchSpy.mock.calls.find(([input]) => /\/decks\/\d+\/items$/.test(String(input)));
	if (!call) return undefined;
	return { url: String(call[0]), body: JSON.parse(String(call[1]?.body)) };
}

describe('AddToDeckModal', () => {
	beforeEach(() => localStorage.clear());
	afterEach(() => vi.restoreAllMocks());

	it('adds a single card to the chosen deck and zone', async () => {
		mockBackend([deck(7, 'Mono Red'), deck(9, 'Elves')]);
		const onAdded = vi.fn();
		const onClose = vi.fn();
		render(AddToDeckModal, { open: true, cards: [sol], onAdded, onClose });

		await page.getByLabelText('Deck').selectOptions('9');
		await page.getByRole('radio', { name: 'Side' }).click();
		await page.getByLabelText('Quantity').fill('2');
		await page.getByRole('button', { name: 'Add to deck' }).click();

		await vi.waitFor(() => expect(onAdded).toHaveBeenCalledWith('Elves'));
		expect(onClose).toHaveBeenCalled();
		expect(postedItems()).toEqual({
			url: '/decks/9/items',
			body: {
				items: [
					{ oracle_id: 'sol', scryfall_id: '', treatment: '', desired_quantity: 2, zone: 'side' }
				]
			}
		});
	});

	it('preselects the last deck used', async () => {
		localStorage.setItem('smc-last-deck-id', '9');
		mockBackend([deck(7, 'Mono Red'), deck(9, 'Elves')]);
		render(AddToDeckModal, { open: true, cards: [sol], onClose: vi.fn() });

		await expect.element(page.getByLabelText('Deck')).toHaveValue('9');
	});

	it('offers a copies toggle in bulk mode and adds one of each', async () => {
		mockBackend([deck(7, 'Mono Red')]);
		const onAdded = vi.fn();
		render(AddToDeckModal, {
			open: true,
			bulk: true,
			cards: [
				{ ...sol, copies: 3 },
				{ oracle_id: 'bolt', scryfall_id: 'm10-1', name: 'Lightning Bolt', copies: 4 }
			],
			onAdded,
			onClose: vi.fn()
		});

		await expect.element(page.getByText('2 cards · 7 copies selected')).toBeInTheDocument();
		await expect.element(page.getByRole('radio', { name: 'All selected (7)' })).toBeChecked();
		await page.getByRole('radio', { name: '1 of each' }).click();
		await page.getByRole('button', { name: 'Add to deck' }).click();

		await vi.waitFor(() => expect(onAdded).toHaveBeenCalledWith('Mono Red'));
		const posted = postedItems();
		expect(posted?.url).toBe('/decks/7/items');
		expect(
			posted?.body.items.map((i: { oracle_id: string; desired_quantity: number }) => [
				i.oracle_id,
				i.desired_quantity
			])
		).toEqual([
			['sol', 1],
			['bolt', 1]
		]);
	});

	it('prompts to create a deck when there are none', async () => {
		mockBackend([]);
		render(AddToDeckModal, { open: true, cards: [sol], onClose: vi.fn() });

		await expect.element(page.getByText('You have no decks yet')).toBeInTheDocument();
		await expect.element(page.getByPlaceholder('New deck name')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Add to deck' })).toBeDisabled();
	});
});
