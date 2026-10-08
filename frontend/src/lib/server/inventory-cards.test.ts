import { describe, expect, it, vi } from 'vitest';
import { inventoryCard, inventoryItem } from '$lib/test/fixtures/inventory';
import { fetchAllInventoryCards } from './inventory-cards';

function page(data: ReturnType<typeof inventoryCard>[], totalPages = 1) {
	return new Response(JSON.stringify({ data, total_pages: totalPages }), {
		headers: { 'content-type': 'application/json' }
	});
}

describe('fetchAllInventoryCards', () => {
	it('merges printings across every page and counts each inventory record once', async () => {
		const box = inventoryItem(1, { storage_location_id: 1, quantity: 2 });
		const binder = inventoryItem(2, { storage_location_id: 2, quantity: 3, treatment: 'foil' });
		const unassigned = inventoryItem(3, { quantity: 4 });
		const otherPrinting = inventoryItem(4, { scryfall_id: 'printing-2' });
		const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input) => {
			const requestedPage = new URL(String(input)).searchParams.get('page');
			if (requestedPage === '1') return page([inventoryCard('printing-1', [box])], 3);
			if (requestedPage === '2') {
				return page(
					[inventoryCard('printing-2', [otherPrinting]), inventoryCard('printing-1', [binder])],
					3
				);
			}
			return page([inventoryCard('printing-1', [box, unassigned])], 3);
		});

		const result = await fetchAllInventoryCards(fetchMock);

		expect(result.map((card) => card.id)).toEqual(['printing-1', 'printing-2']);
		expect(result[0].inventory.this_printing).toEqual([box, binder, unassigned]);
		expect(result[0].inventory.total_quantity).toBe(9);
		expect(result[1].inventory.total_quantity).toBe(1);
		expect(fetchMock).toHaveBeenCalledTimes(3);
		for (const [input] of fetchMock.mock.calls) {
			expect(new URL(String(input)).searchParams.has('storage_location_id')).toBe(false);
		}
	});

	it('continues beyond the first batch of pages', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input) => {
			const number = Number(new URL(String(input)).searchParams.get('page'));
			return page([inventoryCard(`printing-${number}`, [inventoryItem(number)])], 6);
		});

		const result = await fetchAllInventoryCards(fetchMock);

		expect(result.map((card) => card.id)).toEqual(
			Array.from({ length: 6 }, (_, index) => `printing-${index + 1}`)
		);
		expect(fetchMock).toHaveBeenCalledTimes(6);
	});

	it('returns an empty collection when no inventory exists', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockResolvedValue(page([], 0));
		expect(await fetchAllInventoryCards(fetchMock)).toEqual([]);
		expect(fetchMock).toHaveBeenCalledTimes(1);
	});

	it.each([1, 2])(
		'rejects a failure on page %i rather than returning a partial collection',
		async (failedPage) => {
			const fetchMock = vi.fn<typeof fetch>().mockImplementation(async (input) => {
				const number = Number(new URL(String(input)).searchParams.get('page'));
				return number === failedPage
					? new Response(null, { status: 503 })
					: page([inventoryCard('printing-1', [inventoryItem(1)])], 2);
			});

			await expect(fetchAllInventoryCards(fetchMock)).rejects.toThrow(
				'Failed to load all cards (HTTP 503)'
			);
		}
	);

	it('propagates network failures', async () => {
		const fetchMock = vi.fn<typeof fetch>().mockRejectedValue(new Error('Connection refused'));
		await expect(fetchAllInventoryCards(fetchMock)).rejects.toThrow('Connection refused');
	});
});
