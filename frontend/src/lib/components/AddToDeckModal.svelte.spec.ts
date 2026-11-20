import { page } from 'vitest/browser';
import { afterEach, beforeEach, describe, expect, it, vi, type MockInstance } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { notifications, type DeckSummary } from '#lib';
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
		aggregate_shortfall: 0,
		aggregate_over_committed: 0
	};
}

const sol = { oracle_id: 'sol', scryfall_id: 'cmm-1', name: 'Sol Ring', copies: 1 };

let fetchSpy: MockInstance<typeof fetch>;

/**
 * Route the deck list (via the /api proxy) and the add endpoint to canned
 * responses. storedZones is what the add endpoint reports the server stored.
 */
function mockBackend(decks: DeckSummary[], storedZones?: string[]) {
	fetchSpy = vi.spyOn(globalThis, 'fetch').mockImplementation(async (input) => {
		const url = String(input);
		if (url.endsWith('/api/decks')) {
			return new Response(JSON.stringify(decks), { status: 200 });
		}
		if (/\/decks\/\d+\/items$/.test(url)) {
			return new Response(JSON.stringify({ success: true, zones: storedZones }), { status: 200 });
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
		await render(AddToDeckModal, { open: true, cards: [sol], onAdded, onClose });

		await page.getByLabelText('Deck', { exact: true }).selectOptions('9');
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

	it('reports the zone the server stored, e.g. Other for a token', async () => {
		mockBackend([deck(7, 'Rabbits')], ['other']);
		const success = vi.spyOn(notifications, 'success');
		const rabbit = { oracle_id: 'rabbit', scryfall_id: 'tblb-1', name: 'Rabbit', copies: 1 };
		await render(AddToDeckModal, { open: true, cards: [rabbit], onClose: vi.fn() });

		await page.getByRole('button', { name: 'Add to deck' }).click();

		await vi.waitFor(() => expect(success).toHaveBeenCalledWith('Added Rabbit to Rabbits (Other)'));
	});

	it('preselects the last deck used', async () => {
		localStorage.setItem('smc-last-deck-id', '9');
		mockBackend([deck(7, 'Mono Red'), deck(9, 'Elves')]);
		await render(AddToDeckModal, { open: true, cards: [sol], onClose: vi.fn() });

		await expect.element(page.getByLabelText('Deck', { exact: true })).toHaveValue('9');
	});

	it('offers a copies toggle in bulk mode and adds one of each', async () => {
		mockBackend([deck(7, 'Mono Red')]);
		const onAdded = vi.fn();
		await render(AddToDeckModal, {
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
		await render(AddToDeckModal, { open: true, cards: [sol], onClose: vi.fn() });

		await expect
			.element(page.getByText('You have no decks yet. Name your first one:'))
			.toBeInTheDocument();
		await expect.element(page.getByPlaceholder('New deck name')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Add to deck' })).toBeDisabled();
	});
});
