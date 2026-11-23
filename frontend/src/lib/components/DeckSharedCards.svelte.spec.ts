import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { SharedCard } from '#lib';
import DeckSharedCards from './DeckSharedCards.svelte';

const THIS_DECK = 7;

// The Swamp case: two decks pin the German printing more often than it is owned.
const swamp: SharedCard = {
	oracle_id: 'oracle-swamp',
	name: 'Swamp',
	conflicts: [
		{
			scryfall_id: 'scry-de',
			set_code: 'ecl',
			collector_number: '271',
			lang: 'de',
			owned: 11,
			wanted: 15,
			decks: [
				{ deck_id: 3, deck_name: 'Fluch der Fäulnis', quantity: 8 },
				{ deck_id: THIS_DECK, deck_name: 'Lorwyn Eclipsed Elfen', quantity: 7 }
			]
		}
	]
};

// Decks together want more copies of the card than are owned.
const bolt: SharedCard = {
	oracle_id: 'oracle-bolt',
	name: 'Lightning Bolt',
	conflicts: [
		{
			scryfall_id: '',
			owned: 8,
			wanted: 9,
			decks: [
				{ deck_id: 4, deck_name: 'Burn', quantity: 4 },
				{ deck_id: 5, deck_name: 'Prowess', quantity: 5 }
			]
		}
	]
};

describe('DeckSharedCards', () => {
	it('shows one chip per card with the reasons collapsed', async () => {
		await render(DeckSharedCards, { cards: [bolt, swamp], deckId: THIS_DECK });

		await expect
			.element(page.getByRole('button', { name: 'Lightning Bolt' }))
			.toHaveAttribute('aria-expanded', 'false');
		await expect.element(page.getByRole('button', { name: 'Swamp' })).toBeInTheDocument();
		await expect.element(page.getByText(/you own/)).not.toBeInTheDocument();
	});

	it('names the over-pinned printing and the decks behind it', async () => {
		await render(DeckSharedCards, { cards: [swamp], deckId: THIS_DECK });

		const chip = page.getByRole('button', { name: 'Swamp' });
		await chip.click();

		await expect.element(chip).toHaveAttribute('aria-expanded', 'true');
		await expect.element(page.getByText('ECL #271 · German')).toBeInTheDocument();
		await expect
			.element(page.getByText(/is pinned 15× in your decks, you own 11\./))
			.toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Fluch der Fäulnis' }))
			.toHaveAttribute('href', expect.stringMatching(/\/decks\/3$/));
		// The current deck is named, not linked.
		await expect.element(page.getByText('this deck')).toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Lorwyn Eclipsed Elfen' }))
			.not.toBeInTheDocument();
	});

	it('explains a whole-card conflict and switches between cards', async () => {
		await render(DeckSharedCards, { cards: [bolt, swamp], deckId: THIS_DECK });

		await page.getByRole('button', { name: 'Lightning Bolt' }).click();
		await expect.element(page.getByText(/Your decks want 9, you own 8\./)).toBeInTheDocument();
		await expect.element(page.getByRole('link', { name: 'Burn' })).toBeInTheDocument();

		await page.getByRole('button', { name: 'Swamp' }).click();
		await expect.element(page.getByText(/Your decks want/)).not.toBeInTheDocument();
		await expect.element(page.getByText('ECL #271 · German')).toBeInTheDocument();

		await page.getByRole('button', { name: 'Swamp' }).click();
		await expect.element(page.getByText(/you own/)).not.toBeInTheDocument();
	});

	it('shows printed_name on the chip when present', async () => {
		await render(DeckSharedCards, {
			cards: [{ ...swamp, printed_name: 'Sumpf' }],
			deckId: THIS_DECK
		});

		await expect.element(page.getByRole('button', { name: 'Sumpf' })).toBeInTheDocument();
	});
});
