import type { EnhancedCardResult, Inventory } from '$lib';

export function inventoryItem(id: number, overrides: Partial<Inventory> = {}): Inventory {
	return {
		id,
		created_at: '2026-10-08T10:00:00Z',
		updated_at: '2026-10-08T10:00:00Z',
		scryfall_id: 'printing-1',
		oracle_id: 'oracle-1',
		treatment: 'nonfoil',
		quantity: 1,
		...overrides
	};
}

export function inventoryCard(
	id: string,
	items: Inventory[],
	overrides: Partial<EnhancedCardResult> = {}
): EnhancedCardResult {
	return {
		id,
		oracle_id: 'oracle-1',
		name: 'Lightning Bolt',
		set_name: 'Alpha',
		collector_number: '161',
		language: 'en',
		color_identity: ['R'],
		finishes: ['nonfoil', 'foil'],
		prices: { usd: '5.00', usd_foil: '10.00' },
		inventory: {
			this_printing: items,
			other_printings: [],
			total_quantity: items.reduce((total, item) => total + item.quantity, 0)
		},
		...overrides
	};
}
