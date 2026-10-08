import { createRawSnippet } from 'svelte';
import { page } from 'vitest/browser';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { inventoryCard, inventoryItem } from '$lib/test/fixtures/inventory';
import { keyboard } from '$lib/stores/keyboard.svelte';
import { currency } from '$lib/stores/currency.svelte';
import InventoryBrowser from './InventoryBrowser.svelte';

vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

const header = createRawSnippet(() => ({ render: () => '<h1>All Cards</h1>' }));
const cards = [
	inventoryCard('printing-1', [
		inventoryItem(1, {
			storage_location_id: 1,
			storage_location: {
				BaseModel: { id: 1, created_at: '', updated_at: '' },
				name: 'Red Box',
				storage_type: 'Box'
			},
			quantity: 2
		}),
		inventoryItem(2, { storage_location_id: 1, quantity: 3, treatment: 'foil' }),
		inventoryItem(3)
	]),
	inventoryCard('printing-2', [inventoryItem(4)], { name: 'Counterspell', set_name: 'Beta' })
];

describe('InventoryBrowser', () => {
	beforeEach(() => {
		localStorage.clear();
		currency.set('usd');
	});

	it.each([true, false])(
		'shows only owned treatment badges in table view (readOnly=%s)',
		async (readOnly) => {
			const ownedCards = [
				...cards,
				inventoryCard(
					'foil-only',
					[
						inventoryItem(5, { treatment: 'foil' }),
						inventoryItem(6, { treatment: 'nonfoil', quantity: 0 })
					],
					{ name: 'Foil Only' }
				),
				inventoryCard('legacy', [inventoryItem(7, { treatment: '' })], { name: 'Legacy Card' })
			];
			render(InventoryBrowser, { cards: ownedCards, allLocations: [], header, readOnly });
			await page.getByRole('button', { name: 'Table view' }).click();

			const nonfoilOnly = page.getByRole('row').filter({ hasText: 'Counterspell' });
			await expect.element(nonfoilOnly.getByText('Nonfoil', { exact: true })).toBeInTheDocument();
			await expect.element(nonfoilOnly.getByText('Foil', { exact: true })).not.toBeInTheDocument();
			const foilOnly = page.getByRole('row').filter({ hasText: 'Foil Only' });
			await expect.element(foilOnly.getByText('Foil', { exact: true })).toBeInTheDocument();
			await expect.element(foilOnly.getByText('Nonfoil', { exact: true })).not.toBeInTheDocument();
			const both = page.getByRole('row').filter({ hasText: 'Lightning Bolt' });
			await expect.element(both.getByText('Nonfoil', { exact: true })).toBeInTheDocument();
			await expect.element(both.getByText('Foil', { exact: true })).toBeInTheDocument();
			expect(both.getByText('Nonfoil', { exact: true }).elements()).toHaveLength(1);
			const legacy = page.getByRole('row').filter({ hasText: 'Legacy Card' });
			await expect.element(legacy.getByText('Nonfoil', { exact: true })).toBeInTheDocument();
			await expect.element(legacy.getByText('Foil', { exact: true })).not.toBeInTheDocument();
		}
	);

	it('sorts the whole collection before pagination and preserves sorting across views and filters', async () => {
		const manyCards = Array.from({ length: 25 }, (_, index) =>
			inventoryCard(`printing-${index}`, [inventoryItem(index + 1)], {
				name: `Card ${index}`,
				prices: { usd: String(index + 1) }
			})
		);
		render(InventoryBrowser, {
			cards: manyCards,
			allLocations: [],
			header,
			readOnly: true,
			showPriceSort: true
		});
		await page.getByRole('button', { name: '»' }).click();
		await expect.element(page.getByRole('button', { name: 'Page 2 of 2' })).toBeInTheDocument();

		await page.getByRole('combobox', { name: 'Sort by' }).selectOptions('price-desc');
		await expect.element(page.getByRole('button', { name: 'Page 1 of 2' })).toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: /^Card \d+$/ }).first())
			.toHaveTextContent('Card 24');
		await expect
			.element(page.getByRole('link', { name: 'Card 0', exact: true }))
			.not.toBeInTheDocument();

		await page.getByRole('button', { name: 'Table view' }).click();
		await expect
			.element(page.getByRole('link', { name: /^Card \d+$/ }).first())
			.toHaveTextContent('Card 24');
		await expect
			.element(page.getByRole('cell', { name: '$25.00', exact: true }))
			.toBeInTheDocument();
		await page.getByPlaceholder('Filter by name, set, or treatment...').fill('Card 2');
		await expect
			.element(page.getByRole('link', { name: /^Card \d+$/ }).first())
			.toHaveTextContent('Card 24');
		await page.getByRole('combobox', { name: 'Sort by' }).selectOptions('price-asc');
		await expect
			.element(page.getByRole('link', { name: /^Card \d+$/ }).first())
			.toHaveTextContent('Card 2');

		await page.getByRole('combobox', { name: 'Sort by' }).selectOptions('recent');
		await page.getByPlaceholder('Filter by name, set, or treatment...').fill('');
		await expect
			.element(page.getByRole('link', { name: /^Card \d+$/ }).first())
			.toHaveTextContent('Card 0');
	});

	it('reorders prices and updates the table when the preferred currency changes', async () => {
		const pricedCards = [
			inventoryCard('one', [inventoryItem(1)], {
				name: 'Card A',
				prices: { usd: '5.00', eur: '20.00' }
			}),
			inventoryCard('two', [inventoryItem(2, { treatment: 'foil' })], {
				name: 'Card B',
				prices: { usd_foil: '10.00', eur_foil: '2.00' }
			}),
			inventoryCard('three', [inventoryItem(3)], { name: 'Unknown price', prices: {} })
		];
		render(InventoryBrowser, {
			cards: pricedCards,
			allLocations: [],
			header,
			readOnly: true,
			showPriceSort: true
		});
		await page.getByRole('combobox', { name: 'Sort by' }).selectOptions('price-desc');
		await expect
			.element(page.getByRole('link', { name: /^Card [AB]$/ }).first())
			.toHaveTextContent('Card B');
		await page.getByRole('button', { name: 'Table view' }).click();
		currency.set('eur');
		await expect
			.element(page.getByRole('link', { name: /^Card [AB]$/ }).first())
			.toHaveTextContent('Card A');
		await expect
			.element(page.getByRole('cell', { name: '€20.00', exact: true }))
			.toBeInTheDocument();
		await expect.element(page.getByRole('cell', { name: '—', exact: true })).toBeInTheDocument();
	});

	it('shows quantities and location links in the read-only grid without edit controls or shortcuts', async () => {
		render(InventoryBrowser, {
			cards,
			allLocations: [],
			header,
			readOnly: true,
			showLocations: true
		});

		await expect
			.element(page.getByRole('link', { name: 'Red Box' }))
			.toHaveAttribute('href', '/inventory/1');
		await expect.element(page.getByText('· 5 copies')).toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Unassigned' }).first())
			.toHaveAttribute('href', '/inventory/unassigned');
		await expect
			.element(page.getByRole('button', { name: '+', exact: true }))
			.not.toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: '−', exact: true }))
			.not.toBeInTheDocument();
		await expect.element(page.getByRole('checkbox')).not.toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: 'Move copies to another location' }))
			.not.toBeInTheDocument();

		const link = page.getByRole('link', { name: 'Lightning Bolt', exact: true }).element();
		link.closest('.card')!.dispatchEvent(new MouseEvent('mouseenter'));
		expect(keyboard.hoveredActions).toBeNull();
	});

	it('shows aggregate quantities and storage in the read-only table', async () => {
		render(InventoryBrowser, {
			cards,
			allLocations: [],
			header,
			readOnly: true,
			showLocations: true
		});
		await page.getByRole('button', { name: 'Table view' }).click();

		await expect
			.element(page.getByRole('columnheader', { name: 'Storage Locations' }))
			.toBeInTheDocument();
		await expect
			.element(
				page
					.getByRole('row')
					.filter({ hasText: 'Lightning Bolt' })
					.getByRole('cell', { name: '6', exact: true })
			)
			.toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Red Box' }))
			.toHaveAttribute('href', '/inventory/1');
		await expect
			.element(page.getByRole('columnheader', { name: 'Actions' }))
			.not.toBeInTheDocument();
		await expect.element(page.getByRole('checkbox')).not.toBeInTheDocument();
	});

	it('updates the location breakdown after refreshed data without changing the total quantity', async () => {
		const screen = render(InventoryBrowser, {
			cards: [cards[0]],
			allLocations: [],
			header,
			readOnly: true,
			showLocations: true
		});
		await expect.element(page.getByRole('link', { name: 'Red Box' })).toBeInTheDocument();
		await screen.rerender({
			cards: [inventoryCard('printing-1', [inventoryItem(5, { quantity: 6 })])]
		});
		await expect.element(page.getByRole('link', { name: 'Red Box' })).not.toBeInTheDocument();
		await expect.element(page.getByText('· 6 copies')).toBeInTheDocument();
	});

	it('filters the complete collection and resets pagination when the filter changes', async () => {
		const manyCards = Array.from({ length: 25 }, (_, index) =>
			inventoryCard(`printing-${index}`, [inventoryItem(index + 1)], { name: `Card ${index}` })
		);
		render(InventoryBrowser, { cards: manyCards, allLocations: [], header, readOnly: true });
		await expect.element(page.getByRole('button', { name: 'Page 1 of 2' })).toBeInTheDocument();
		await page.getByRole('button', { name: '»' }).click();
		await expect
			.element(page.getByRole('link', { name: 'Card 24', exact: true }))
			.toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Card 0', exact: true }))
			.not.toBeInTheDocument();

		await page.getByPlaceholder('Filter by name, set, or treatment...').fill('Card 0');
		await expect
			.element(page.getByRole('link', { name: 'Card 0', exact: true }))
			.toBeInTheDocument();
		await expect.element(page.getByText('Showing 1 of 25 printings')).toBeInTheDocument();
		await expect.element(page.getByRole('button', { name: 'Page 2 of 2' })).not.toBeInTheDocument();
	});

	it('filters by set and by owned treatments beyond the first finish', async () => {
		render(InventoryBrowser, { cards, allLocations: [], header, readOnly: true });
		const filter = page.getByPlaceholder('Filter by name, set, or treatment...');
		await filter.fill('Beta');
		await expect
			.element(page.getByRole('link', { name: 'Counterspell', exact: true }))
			.toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Lightning Bolt', exact: true }))
			.not.toBeInTheDocument();
		await filter.fill('Foil');
		await expect
			.element(page.getByRole('link', { name: 'Lightning Bolt', exact: true }))
			.toBeInTheDocument();
	});

	it('shows a load error rather than an empty collection', async () => {
		render(InventoryBrowser, {
			cards: [],
			allLocations: [],
			header,
			readOnly: true,
			error: 'Failed to load all cards'
		});
		await expect.element(page.getByText('Failed to load all cards')).toBeInTheDocument();
		await expect.element(page.getByText('No cards found')).not.toBeInTheDocument();
	});

	it('shows an empty state for an empty collection', async () => {
		render(InventoryBrowser, { cards: [], allLocations: [], header, readOnly: true });
		await expect.element(page.getByText('No cards found')).toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'Search for Cards' }))
			.toHaveAttribute('href', '/search');
	});

	it('keeps edit controls available by default', async () => {
		const screen = render(InventoryBrowser, { cards: [cards[0]], allLocations: [], header });
		await expect
			.element(page.getByRole('button', { name: '+', exact: true }).first())
			.toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: '−', exact: true }).first())
			.toBeInTheDocument();
		await expect.element(page.getByRole('checkbox').first()).toBeInTheDocument();
		await expect
			.element(page.getByRole('button', { name: 'Move copies to another location' }).first())
			.toBeInTheDocument();
		const link = page.getByRole('link', { name: 'Lightning Bolt', exact: true }).element();
		link.closest('.card')!.dispatchEvent(new MouseEvent('mouseenter'));
		expect(keyboard.hoveredActions).not.toBeNull();
		await screen.unmount();
		expect(keyboard.hoveredActions).toBeNull();
	});
});
