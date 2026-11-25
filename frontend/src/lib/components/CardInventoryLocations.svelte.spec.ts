import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { inventoryItem } from '#lib/test/fixtures/inventory.ts';
import type { Inventory } from '#lib';
import CardInventoryLocations from './CardInventoryLocations.svelte';

function makeLocation(
	id: number,
	name: string,
	sortOrder: number
): NonNullable<Inventory['storage_location']> {
	return {
		BaseModel: {
			id,
			created_at: '2026-10-08T10:00:00Z',
			updated_at: '2026-10-08T10:00:00Z'
		},
		name,
		storage_type: 'Box',
		sort_order: sortOrder
	};
}

describe('CardInventoryLocations', () => {
	it('orders assigned locations by sort_order and places Unassigned last', async () => {
		const inventory = [
			inventoryItem(1, { storage_location_id: undefined, quantity: 1 }),
			inventoryItem(2, {
				storage_location_id: 1,
				storage_location: makeLocation(1, 'Third Box', 30),
				quantity: 2
			}),
			inventoryItem(3, {
				storage_location_id: 2,
				storage_location: makeLocation(2, 'First Binder', 10),
				quantity: 1
			}),
			inventoryItem(4, {
				storage_location_id: 3,
				storage_location: makeLocation(3, 'Second Box', 20),
				quantity: 3
			})
		];

		await render(CardInventoryLocations, { inventory });

		const links = page.getByRole('list', { name: 'Storage locations' }).getByRole('link');
		const elements = links.elements();
		expect(elements.map((el) => el.textContent?.trim())).toEqual([
			'First Binder',
			'Second Box',
			'Third Box',
			'Unassigned'
		]);
	});

	it('breaks sort_order ties by location id ascending', async () => {
		const inventory = [
			inventoryItem(1, {
				storage_location_id: 5,
				storage_location: makeLocation(5, 'Box Five', 10),
				quantity: 1
			}),
			inventoryItem(2, {
				storage_location_id: 2,
				storage_location: makeLocation(2, 'Box Two', 10),
				quantity: 2
			})
		];

		await render(CardInventoryLocations, { inventory });

		const links = page.getByRole('list', { name: 'Storage locations' }).getByRole('link');
		expect(links.elements().map((el) => el.textContent?.trim())).toEqual(['Box Two', 'Box Five']);
	});
});
