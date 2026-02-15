/**
 * Data adapters for normalizing different card data sources into DisplayableCard
 */

import type { EnhancedCardResult, EnrichedListItem } from '$lib';
import type { DisplayableCard, CardSourceType } from './types';

/**
 * Adapt EnhancedCardResult (from inventory cards endpoint) to DisplayableCard
 */
export function fromInventoryCard(card: EnhancedCardResult): DisplayableCard {
	return {
		id: card.id,
		scryfallId: card.id,
		oracleId: card.oracle_id,
		name: card.name,
		setName: card.set_name,
		collectorNumber: card.collector_number,
		imageUri: card.image_uri,
		treatment: card.finishes[0] ?? 'nonfoil',
		finishes: card.finishes,
		frameEffects: card.frame_effects,
		quantity: card.inventory.total_quantity,
		prices: card.prices,
		_source: card,
		_sourceType: 'inventory'
	};
}

/**
 * Adapt EnhancedCardResult (from search endpoint) to DisplayableCard
 */
export function fromSearchResult(card: EnhancedCardResult): DisplayableCard {
	return {
		id: card.id,
		scryfallId: card.id,
		oracleId: card.oracle_id,
		name: card.name,
		setName: card.set_name,
		collectorNumber: card.collector_number,
		imageUri: card.image_uri,
		treatment: card.finishes[0] ?? 'nonfoil',
		finishes: card.finishes,
		frameEffects: card.frame_effects,
		quantity: card.inventory.total_quantity,
		prices: card.prices,
		_source: card,
		_sourceType: 'search'
	};
}

/**
 * Adapt EnrichedListItem to DisplayableCard
 */
export function fromListItem(item: EnrichedListItem): DisplayableCard {
	return {
		id: item.id,
		scryfallId: item.scryfall_id,
		oracleId: item.oracle_id,
		name: item.name ?? 'Unknown Card',
		setName: item.set_name,
		setCode: item.set_code,
		collectorNumber: item.collector_number,
		rarity: item.rarity,
		imageUri: undefined, // List items don't have image URIs - we'll need to fetch or construct
		treatment: item.treatment,
		finishes: [item.treatment],
		quantity: item.collected_quantity,
		secondaryQuantity: item.desired_quantity,
		prices: item.current_price ? { usd: item.current_price.toString() } : undefined,
		_source: item,
		_sourceType: 'list'
	};
}

/**
 * Batch adapt EnhancedCardResult array
 */
export function adaptInventoryCards(cards: EnhancedCardResult[]): DisplayableCard[] {
	return cards.map(fromInventoryCard);
}

/**
 * Batch adapt search results
 */
export function adaptSearchResults(cards: EnhancedCardResult[]): DisplayableCard[] {
	return cards.map(fromSearchResult);
}

/**
 * Batch adapt list items
 */
export function adaptListItems(items: EnrichedListItem[]): DisplayableCard[] {
	return items.map(fromListItem);
}

/**
 * Get the original source data with proper typing
 */
export function getSourceData<T extends CardSourceType>(
	card: DisplayableCard,
	expectedType: T
): T extends 'inventory' | 'search'
	? EnhancedCardResult
	: T extends 'list'
		? EnrichedListItem
		: never {
	if (card._sourceType !== expectedType) {
		throw new Error(`Expected source type ${expectedType}, got ${card._sourceType}`);
	}
	return card._source as ReturnType<typeof getSourceData<T>>;
}

/**
 * Check if a card is from a specific source type
 */
export function isInventoryCard(
	card: DisplayableCard
): card is DisplayableCard & { _sourceType: 'inventory' } {
	return card._sourceType === 'inventory';
}

export function isSearchResult(
	card: DisplayableCard
): card is DisplayableCard & { _sourceType: 'search' } {
	return card._sourceType === 'search';
}

export function isListItem(
	card: DisplayableCard
): card is DisplayableCard & { _sourceType: 'list' } {
	return card._sourceType === 'list';
}

/**
 * Determine if a treatment represents a foil card
 */
export function isFoilTreatment(treatment: string): boolean {
	const lowerTreatment = treatment.toLowerCase();
	return lowerTreatment !== 'nonfoil' && lowerTreatment !== 'glossy';
}
