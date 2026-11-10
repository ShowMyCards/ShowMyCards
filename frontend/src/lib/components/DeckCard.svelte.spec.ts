import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { DeckSummary } from '$lib';
import DeckCard from './DeckCard.svelte';

function makeDeck(overrides: Partial<DeckSummary> = {}): DeckSummary {
	return {
		id: 1,
		created_at: '',
		updated_at: '',
		name: 'Byrke Commander',
		description: '',
		total_items: 1,
		total_cards_demand: 1,
		aggregate_shortfall: 0,
		aggregate_over_committed: 0,
		...overrides
	};
}

describe('DeckCard ownership badges', () => {
	it('says "Fully owned" when nothing is short or shared', async () => {
		render(DeckCard, { deck: makeDeck() });

		await expect.element(page.getByText('Fully owned')).toBeInTheDocument();
		await expect.element(page.getByText(/Shared \d/)).not.toBeInTheDocument();
	});

	it('says "Shared N" instead of "Fully owned" when other decks want the same cards', async () => {
		render(DeckCard, { deck: makeDeck({ aggregate_over_committed: 2 }) });

		await expect.element(page.getByText('Shared 2')).toBeInTheDocument();
		await expect.element(page.getByText('Fully owned')).not.toBeInTheDocument();
	});

	it('says "Short N" when the deck needs cards that are not owned', async () => {
		render(DeckCard, { deck: makeDeck({ aggregate_shortfall: 3 }) });

		await expect.element(page.getByText('Short 3')).toBeInTheDocument();
		await expect.element(page.getByText('Fully owned')).not.toBeInTheDocument();
		await expect.element(page.getByText(/Shared \d/)).not.toBeInTheDocument();
	});

	it('shows both "Short N" and "Shared N" when both apply', async () => {
		render(DeckCard, { deck: makeDeck({ aggregate_shortfall: 1, aggregate_over_committed: 2 }) });

		await expect.element(page.getByText('Short 1')).toBeInTheDocument();
		await expect.element(page.getByText('Shared 2')).toBeInTheDocument();
		await expect.element(page.getByText('Fully owned')).not.toBeInTheDocument();
	});

	it('shows no ownership badge for an empty deck', async () => {
		render(DeckCard, { deck: makeDeck({ total_items: 0, total_cards_demand: 0 }) });

		await expect.element(page.getByText('Fully owned')).not.toBeInTheDocument();
		await expect.element(page.getByText(/Short \d/)).not.toBeInTheDocument();
		await expect.element(page.getByText(/Shared \d/)).not.toBeInTheDocument();
	});
});
