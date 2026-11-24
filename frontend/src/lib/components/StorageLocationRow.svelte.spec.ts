import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { StorageLocation } from '#lib';
import StorageLocationRow from './StorageLocationRow.svelte';

function makeLocation(overrides: Partial<StorageLocation> = {}): StorageLocation {
	return {
		id: 1,
		created_at: '2026-10-08T10:00:00Z',
		updated_at: '2026-10-08T10:00:00Z',
		name: 'Main Box',
		storage_type: 'Box',
		sort_order: 1,
		...overrides
	};
}

describe('StorageLocationRow', () => {
	it('disables Move up on the first row and enables Move down', async () => {
		await render(StorageLocationRow, {
			location: makeLocation({ name: 'First Box' }),
			index: 0,
			total: 3,
			onDrop: vi.fn(),
			onMoveUp: vi.fn(),
			onMoveDown: vi.fn()
		});

		await expect.element(page.getByRole('button', { name: 'Move First Box up' })).toBeDisabled();
		await expect.element(page.getByRole('button', { name: 'Move First Box down' })).toBeEnabled();
	});

	it('disables Move down on the last row and enables Move up', async () => {
		await render(StorageLocationRow, {
			location: makeLocation({ name: 'Last Box' }),
			index: 2,
			total: 3,
			onDrop: vi.fn(),
			onMoveUp: vi.fn(),
			onMoveDown: vi.fn()
		});

		await expect.element(page.getByRole('button', { name: 'Move Last Box up' })).toBeEnabled();
		await expect.element(page.getByRole('button', { name: 'Move Last Box down' })).toBeDisabled();
	});

	it('invokes onMoveUp and onMoveDown on middle rows', async () => {
		const onMoveUp = vi.fn();
		const onMoveDown = vi.fn();
		await render(StorageLocationRow, {
			location: makeLocation({ name: 'Middle Box' }),
			index: 1,
			total: 3,
			onDrop: vi.fn(),
			onMoveUp,
			onMoveDown
		});

		await page.getByRole('button', { name: 'Move Middle Box up' }).click();
		await page.getByRole('button', { name: 'Move Middle Box down' }).click();

		expect(onMoveUp).toHaveBeenCalledTimes(1);
		expect(onMoveDown).toHaveBeenCalledTimes(1);
	});

	it('hides reorder controls while editing a row', async () => {
		await render(StorageLocationRow, {
			location: makeLocation({ name: 'Editable Box' }),
			index: 1,
			total: 3,
			onDrop: vi.fn(),
			onMoveUp: vi.fn(),
			onMoveDown: vi.fn()
		});

		await page.getByRole('button', { name: 'Edit this location' }).click();

		await expect
			.element(page.getByRole('button', { name: 'Move Editable Box up' }))
			.not.toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: 'Move Editable Box down' }))
			.not.toBeInTheDocument();
	});
});
