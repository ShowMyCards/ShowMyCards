/**
 * Runtime-accurate type helpers
 *
 * Go's JSON marshaling flattens embedded structs, but tygo generates nested types.
 * These helpers provide types that match the actual API responses.
 */

import type { BaseModel, StorageLocation, Inventory, SortingRule, Job } from './models';

/**
 * Helper type to flatten embedded BaseModel into parent type
 * Use this for actual API responses where Go flattens embedded structs
 */
export type Flattened<T extends { BaseModel: BaseModel }> = Omit<T, 'BaseModel'> & BaseModel;

// Runtime-accurate types that match actual API responses
export type RuntimeStorageLocation = Flattened<StorageLocation>;
export type RuntimeInventory = Flattened<Inventory>;
export type RuntimeSortingRule = Flattened<SortingRule>;
export type RuntimeJob = Flattened<Job>;

/**
 * Scryfall API Card Type
 * Represents a card object from the Scryfall API
 * @see https://scryfall.com/docs/api/cards
 */
export interface ScryfallCard {
	id: string;
	oracle_id: string;
	name: string;
	mana_cost?: string;
	cmc?: number;
	type_line: string;
	oracle_text?: string;
	colors?: string[];
	color_identity?: string[];
	set: string;
	set_name: string;
	collector_number: string;
	rarity: string;
	prices: {
		usd?: string | null;
		usd_foil?: string | null;
		eur?: string | null;
		eur_foil?: string | null;
		tix?: string | null;
	};
	finishes: string[];
	frame_effects?: string[];
	image_uris?: {
		thumb?: string;
		grid?: string;
		display?: string;
		crop?: string;
		art?: string;
		jpeg?: string;
		/** Not hotlinkable after 2027-01-01; use a WebP version for display. */
		png?: string;
		/** @deprecated Removed by Scryfall after 2027-01-01. Use `thumb`. */
		small?: string;
		/** @deprecated Renamed by Scryfall to `jpeg`. Use `grid` or `jpeg`. */
		normal?: string;
		/** @deprecated Removed by Scryfall after 2027-01-01. Use `display`. */
		large?: string;
		/** @deprecated Removed by Scryfall after 2027-01-01. Use `art`. */
		art_crop?: string;
		/** @deprecated Removed by Scryfall after 2027-01-01. Use `crop`. */
		border_crop?: string;
	};
	card_faces?: Array<{
		name: string;
		type_line: string;
		oracle_text?: string;
		mana_cost?: string;
		colors?: string[];
		image_uris?: ScryfallCard['image_uris'];
	}>;
	// Allow other Scryfall fields we don't explicitly type
	[key: string]: unknown;
}

/**
 * Form data for creating/updating list items
 */
export interface ListItemData {
	scryfall_id: string;
	treatment: string;
	quantity: number;
	notes?: string;
}

/**
 * Search form result structure
 */
export interface SearchFormResult {
	query: string;
	searchResults: ScryfallCard[];
}
