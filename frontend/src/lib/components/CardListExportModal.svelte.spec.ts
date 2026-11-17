import { page, userEvent } from 'vitest/browser';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { CardListExport } from '#lib/utils/card-list-export.ts';
import CardListExportModal from './CardListExportModal.svelte';

const all: CardListExport = {
	lines: [
		{ quantity: 1, name: 'Fire // Ice' },
		{ quantity: 4, name: 'Lightning Bolt' }
	],
	skipped: 0
};
const missing: CardListExport = { lines: [{ quantity: 2, name: 'Lightning Bolt' }], skipped: 0 };
const pinned: CardListExport = {
	lines: [
		{ quantity: 1, name: 'Dark Confidant', expansion: 'Modern Masters' },
		{ quantity: 2, name: 'Lightning Bolt' }
	],
	skipped: 0
};
const nothingMissing: CardListExport = { lines: [], skipped: 0 };

function exported() {
	return page.getByRole('textbox', { name: 'Exported card list' });
}

describe('CardListExportModal', () => {
	afterEach(() => vi.restoreAllMocks());

	it('opens on the missing cards', async () => {
		await render(CardListExportModal, { open: true, onClose: () => {}, all, missing });

		await expect.element(page.getByRole('radio', { name: 'Only missing' })).toBeChecked();
		await expect.element(exported()).toHaveValue('2 Lightning Bolt');
		await expect.element(page.getByText('1 line · 2 cards.', { exact: false })).toBeVisible();
	});

	it('shows pinned expansions in Cardmarket syntax', async () => {
		await render(CardListExportModal, { open: true, onClose: () => {}, all, missing: pinned });

		await expect
			.element(exported())
			.toHaveValue('1 Dark Confidant (Modern Masters)\n2 Lightning Bolt');
		await expect
			.element(page.getByText('Formatted for Cardmarket Wants lists', { exact: false }))
			.toBeVisible();
	});

	it('switches to every card', async () => {
		await render(CardListExportModal, { open: true, onClose: () => {}, all, missing });

		await page.getByRole('radio', { name: 'All cards' }).click();

		await expect.element(exported()).toHaveValue('1 Fire // Ice\n4 Lightning Bolt');
	});

	it('copies the shown text to the clipboard', async () => {
		const writeText = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue();
		await render(CardListExportModal, { open: true, onClose: () => {}, all, missing });

		await page.getByRole('button', { name: 'Copy' }).click();

		expect(writeText).toHaveBeenCalledWith('2 Lightning Bolt');
	});

	it('shows an empty state and disables Copy when nothing is missing', async () => {
		await render(CardListExportModal, {
			open: true,
			onClose: () => {},
			all,
			missing: nothingMissing
		});

		await expect.element(page.getByText('Nothing missing', { exact: false })).toBeVisible();
		await expect.element(exported()).not.toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Copy' })).toBeDisabled();
	});

	it('says how many items were left out for missing card data', async () => {
		await render(CardListExportModal, {
			open: true,
			onClose: () => {},
			all,
			missing: { ...missing, skipped: 2 }
		});

		await expect.element(page.getByText('2 items were left out', { exact: false })).toBeVisible();
	});

	it('reopens on the missing cards after closing', async () => {
		const onClose = vi.fn();
		const { rerender } = await render(CardListExportModal, { open: true, onClose, all, missing });

		await page.getByRole('radio', { name: 'All cards' }).click();
		await userEvent.click(page.getByRole('button', { name: 'Close', exact: true }));
		expect(onClose).toHaveBeenCalled();

		await rerender({ open: false });
		await rerender({ open: true });

		await expect.element(page.getByRole('radio', { name: 'Only missing' })).toBeChecked();
		await expect.element(exported()).toHaveValue('2 Lightning Bolt');
	});
});
