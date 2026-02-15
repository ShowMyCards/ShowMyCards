/**
 * Runtime-accurate API types
 *
 * Extends/overrides generated API types to use runtime-accurate model types
 */

import type { RuntimeInventory, RuntimeStorageLocation } from './runtime';
import type { StorageType } from './models';
import type * as Generated from './api';

// Override CardInventoryData to use runtime Inventory type
export interface CardInventoryData {
	this_printing: RuntimeInventory[];
	other_printings: RuntimeInventory[];
	total_quantity: number;
}

// Override EnhancedCardResult to use runtime CardInventoryData
export interface EnhancedCardResult extends Omit<Generated.EnhancedCardResult, 'inventory'> {
	inventory: CardInventoryData;
}

// Override EvaluateResponse to use runtime StorageLocation
export interface EvaluateResponse extends Omit<Generated.EvaluateResponse, 'storage_location'> {
	matched: boolean;
	storage_location?: RuntimeStorageLocation;
	error?: string;
}

// Storage location with card count
export interface StorageLocationWithCount {
	id: number;
	created_at: string;
	updated_at: string;
	name: string;
	storage_type: StorageType;
	card_count: number;
}

// Inventory cards response with runtime-accurate card results
export interface InventoryCardsResponse extends Omit<Generated.InventoryCardsResponse, 'data'> {
	data: EnhancedCardResult[];
}

// Override SearchResponse to use runtime EnhancedCardResult
export interface SearchResponse extends Omit<Generated.SearchResponse, 'data'> {
	data: EnhancedCardResult[];
}

// Re-export other generated types as-is
export type { CardResult, CardPrices, EvaluateRequest } from './api';
