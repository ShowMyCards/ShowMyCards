import type { CardPrices, EnhancedCardResult } from '$lib';
import type { Currency } from '$lib/stores/currency.svelte';

/** Select the same treatment price for display and sorting. */
export function getTreatmentPrice(
	prices: CardPrices,
	treatment: string,
	currency: Currency
): string | undefined {
	if (treatment === 'foil') return currency === 'eur' ? prices.eur_foil : prices.usd_foil;
	// Scryfall has no separate EUR etched price; use EUR foil as in the price display.
	if (treatment === 'etched') return currency === 'eur' ? prices.eur_foil : prices.usd_etched;
	return currency === 'eur' ? prices.eur : prices.usd;
}

/** Highest known unit price among treatments owned for this printing, independent of quantity. */
export function getInventoryUnitPrice(
	card: EnhancedCardResult,
	currency: Currency
): number | undefined {
	let highest: number | undefined;
	for (const item of card.inventory.this_printing) {
		if (item.quantity <= 0) continue;
		const raw = getTreatmentPrice(card.prices, item.treatment || 'nonfoil', currency);
		if (!raw?.trim()) continue;
		const price = Number(raw);
		if (Number.isFinite(price) && price >= 0 && (highest === undefined || price > highest)) {
			highest = price;
		}
	}
	return highest;
}

/** Sort numerically, keeping unavailable prices last and ties in their original order. */
export function sortInventoryCardsByPrice(
	cards: EnhancedCardResult[],
	currency: Currency,
	direction: 'asc' | 'desc'
): EnhancedCardResult[] {
	return cards
		.map((card) => ({ card, price: getInventoryUnitPrice(card, currency) }))
		.sort((a, b) => {
			if (a.price === undefined) return b.price === undefined ? 0 : 1;
			if (b.price === undefined) return -1;
			return direction === 'asc' ? a.price - b.price : b.price - a.price;
		})
		.map(({ card }) => card);
}
