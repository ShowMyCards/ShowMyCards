import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { inventoryCard, inventoryItem } from '#lib/test/fixtures/inventory.ts';
import type { Inventory } from '#lib';
import Page from './+page.svelte';
import type { PageData } from './$types';

// #lib re-exports and Kit's runtime import refreshAll and afterNavigate from this module.
vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));

function makeLocation(
	id: number,
	name: string,
	sortOrder: number,
	storageType: 'Box' | 'Binder' = 'Box'
): NonNullable<Inventory['storage_location']> {
	return {
		BaseModel: {
			id,
			created_at: '2026-10-08T10:00:00Z',
			updated_at: '2026-10-08T10:00:00Z'
		},
		name,
		storage_type: storageType,
		sort_order: sortOrder
	};
}

function makeData(items: Inventory[]): PageData {
	return {
		card: inventoryCard('card-1', items),
		otherPrintings: [],
		deckUsage: []
	} as PageData;
}

describe('/cards/[id]/+page.svelte', () => {
	it('orders storage location badges by sort_order and omits unassigned rows', async () => {
		const items = [
			inventoryItem(1, { storage_location_id: undefined, quantity: 1 }),
			inventoryItem(2, {
				storage_location_id: 1,
				storage_location: makeLocation(1, 'Third Box', 30, 'Box'),
				treatment: 'nonfoil',
				quantity: 2
			}),
			inventoryItem(3, {
				storage_location_id: 2,
				storage_location: makeLocation(2, 'First Binder', 10, 'Binder'),
				treatment: 'foil',
				quantity: 1
			}),
			inventoryItem(4, {
				storage_location_id: 3,
				storage_location: makeLocation(3, 'Second Box', 20, 'Box'),
				treatment: 'nonfoil',
				quantity: 3
			})
		];

		await render(Page, { data: makeData(items) });

		const badges = page.getByRole('list', { name: 'Storage locations' }).getByRole('listitem');
		expect(badges.elements().map((el) => el.textContent?.trim())).toEqual([
			'First Binder (1x foil)',
			'Second Box (3x nonfoil)',
			'Third Box (2x nonfoil)'
		]);
	});

	it('breaks sort_order ties by locationId then treatment, consolidating duplicate rows', async () => {
		const loc5 = makeLocation(5, 'Box Five', 10);
		const loc2 = makeLocation(2, 'Box Two', 10);

		const items = [
			inventoryItem(1, {
				storage_location_id: 5,
				storage_location: loc5,
				treatment: 'nonfoil',
				quantity: 1
			}),
			inventoryItem(2, {
				storage_location_id: 2,
				storage_location: loc2,
				treatment: 'nonfoil',
				quantity: 2
			}),
			inventoryItem(3, {
				storage_location_id: 2,
				storage_location: loc2,
				treatment: '',
				quantity: 1
			}),
			inventoryItem(4, {
				storage_location_id: 2,
				storage_location: loc2,
				treatment: 'foil',
				quantity: 1
			})
		];

		await render(Page, { data: makeData(items) });

		const badges = page.getByRole('list', { name: 'Storage locations' }).getByRole('listitem');
		expect(badges.elements().map((el) => el.textContent?.trim())).toEqual([
			'Box Two (1x foil)',
			'Box Two (3x nonfoil)',
			'Box Five (1x nonfoil)'
		]);
	});
});
