import { describe, expect, it } from 'vitest';
import { compareStorageLocations, type StorageLocationSortable } from './sort-storage';

describe('compareStorageLocations', () => {
	it('orders assigned locations by sortOrder ascending', () => {
		const items: StorageLocationSortable[] = [
			{ id: 1, sortOrder: 30 },
			{ id: 2, sortOrder: 10 },
			{ id: 3, sortOrder: 20 }
		];

		expect([...items].sort(compareStorageLocations).map((i) => i.id)).toEqual([2, 3, 1]);
	});

	it('places unassigned entries last regardless of sortOrder', () => {
		const items: StorageLocationSortable[] = [
			{ id: null, sortOrder: 0 },
			{ id: 2, sortOrder: 20 },
			{ id: 1, sortOrder: undefined },
			{ id: 3, sortOrder: 10 }
		];

		expect([...items].sort(compareStorageLocations).map((i) => i.id)).toEqual([3, 2, 1, null]);
	});

	it('breaks sortOrder ties by id ascending and then treatment lexicographically', () => {
		const items: StorageLocationSortable[] = [
			{ id: 5, sortOrder: 10, treatment: 'nonfoil' },
			{ id: 2, sortOrder: 10, treatment: 'nonfoil' },
			{ id: 2, sortOrder: 10, treatment: 'foil' },
			{ id: 2, sortOrder: 10, treatment: 'etched' }
		];

		expect([...items].sort(compareStorageLocations)).toEqual([
			{ id: 2, sortOrder: 10, treatment: 'etched' },
			{ id: 2, sortOrder: 10, treatment: 'foil' },
			{ id: 2, sortOrder: 10, treatment: 'nonfoil' },
			{ id: 5, sortOrder: 10, treatment: 'nonfoil' }
		]);
	});

	it('returns 0 for equivalent entries', () => {
		expect(compareStorageLocations({ id: 2, sortOrder: 10 }, { id: 2, sortOrder: 10 })).toBe(0);
		expect(compareStorageLocations({ id: null }, { id: null })).toBe(0);
	});
});
