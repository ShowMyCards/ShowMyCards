import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { FormatLegality } from '#lib';
import DeckLegality from './DeckLegality.svelte';

function format(key: string, name: string, reasons: string[] = []): FormatLegality {
	return { format: key, name, legal: reasons.length === 0, reasons };
}

const formats = [
	format('standard', 'Standard', ['Banned: Black Lotus']),
	format('modern', 'Modern'),
	format('commander', 'Commander', ['No commander', 'Deck has 60 cards (needs exactly 100)']),
	format('oathbreaker', 'Oathbreaker', [
		'Command zone needs one planeswalker and one instant or sorcery'
	])
];

describe('DeckLegality', () => {
	it('shows formats in the given order', async () => {
		const screen = await render(DeckLegality, { formats, empty: false });

		const text = screen.container.textContent ?? '';
		const order = ['Standard', 'Modern', 'Commander', 'Oathbreaker'].map((s) => text.indexOf(s));
		expect(order).toEqual([...order].sort((a, b) => a - b));
		expect(order).not.toContain(-1);
	});

	it('shows legal formats as labels and not-legal formats as buttons', async () => {
		await render(DeckLegality, { formats, empty: false });

		await expect.element(page.getByText('Modern (legal)')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Modern' })).not.toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: 'Standard (not legal)' }))
			.toHaveAttribute('aria-expanded', 'false');
	});

	it('expands one format at a time and collapses on a second click', async () => {
		await render(DeckLegality, { formats, empty: false });
		const commander = page.getByRole('button', { name: 'Commander (not legal)' });
		const standard = page.getByRole('button', { name: 'Standard (not legal)' });

		await commander.click();
		await expect.element(commander).toHaveAttribute('aria-expanded', 'true');
		await expect.element(page.getByText('Deck has 60 cards (needs exactly 100)')).toBeVisible();

		await standard.click();
		await expect.element(page.getByText('Banned: Black Lotus')).toBeVisible();
		await expect
			.element(page.getByText('Deck has 60 cards (needs exactly 100)'))
			.not.toBeInTheDocument();
		await expect.element(commander).toHaveAttribute('aria-expanded', 'false');

		await standard.click();
		await expect.element(page.getByText('Banned: Black Lotus')).not.toBeInTheDocument();
	});

	it('closes the reasons when the open format becomes legal', async () => {
		const screen = await render(DeckLegality, { formats, empty: false });
		await page.getByRole('button', { name: 'Standard (not legal)' }).click();
		await expect.element(page.getByText('Banned: Black Lotus')).toBeVisible();

		await screen.rerender({
			formats: [format('standard', 'Standard'), ...formats.slice(1)]
		});

		await expect.element(page.getByText('Banned: Black Lotus')).not.toBeInTheDocument();
		await expect.element(page.getByText('Standard (legal)')).toBeInTheDocument();
	});

	it('asks for cards when the deck is empty', async () => {
		await render(DeckLegality, { formats, empty: true });

		await expect.element(page.getByText('Add cards to check format legality.')).toBeVisible();
		await expect.element(page.getByText('Modern (legal)')).not.toBeInTheDocument();
	});

	it('shows an error when the check failed', async () => {
		await render(DeckLegality, { formats: null, empty: false });

		await expect
			.element(page.getByRole('alert'))
			.toHaveTextContent("Couldn't check format legality.");
	});
});
