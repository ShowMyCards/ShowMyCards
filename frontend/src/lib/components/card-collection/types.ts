import type { Snippet } from 'svelte';
import type { RuntimeStorageLocation as StorageLocation } from '$lib/types/runtime';
import type { CardPrices } from '$lib/types/api';

// Re-export for convenience
export type { StorageLocation };

/**
 * View modes supported by the collection
 */
export type ViewMode = 'grid' | 'table';

/**
 * Source type discriminator for DisplayableCard
 */
export type CardSourceType = 'inventory' | 'search' | 'list';

/**
 * Common interface for displayable card items
 * Normalizes different data sources (inventory, search, list) into a single format
 */
export interface DisplayableCard {
	/** Unique identifier for the item (inventory.id, listItem.id, scryfall_id) */
	id: string | number;
	/** Scryfall ID for card image and details */
	scryfallId: string;
	/** Oracle ID for grouping printings */
	oracleId: string;
	/** Card name */
	name: string;
	/** Set name */
	setName?: string;
	/** Set code */
	setCode?: string;
	/** Collector number */
	collectorNumber?: string;
	/** Card rarity */
	rarity?: string;
	/** Card image URL */
	imageUri?: string;
	/** Card treatment (foil, nonfoil, etched) */
	treatment: string;
	/** Available finishes for this card */
	finishes: string[];
	/** Frame effects for treatment display */
	frameEffects?: string[];
	/** Quantity (meaning varies by context) */
	quantity: number;
	/** Secondary quantity (for lists: desired vs collected) */
	secondaryQuantity?: number;
	/** Price information */
	prices?: CardPrices;
	/** Storage location (for inventory items) */
	storageLocation?: StorageLocation;
	/** Raw source data for context-specific rendering */
	_source: unknown;
	/** Source type discriminator */
	_sourceType: CardSourceType;
}

/**
 * Pagination state
 */
export interface PaginationState {
	currentPage: number;
	totalPages: number;
	pageSize: number;
	totalItems: number;
}

/**
 * Column definition for table view
 */
export interface TableColumn {
	/** Unique column key */
	key: string;
	/** Column header */
	header: string;
	/** Width class (Tailwind) */
	width?: string;
	/** Whether column is sortable */
	sortable?: boolean;
	/** Custom cell renderer snippet */
	render?: Snippet<[DisplayableCard]>;
	/** Accessor function for default rendering */
	accessor?: (item: DisplayableCard) => string | number | undefined;
}

/**
 * Main CardCollection component props
 */
export interface CardCollectionProps {
	/** Items to display (already adapted to DisplayableCard) */
	items: DisplayableCard[];

	/** Loading state */
	loading?: boolean;

	/** Current view mode */
	viewMode?: ViewMode;

	/** Callback when view mode changes */
	onViewModeChange?: (mode: ViewMode) => void;

	/** Pagination state (undefined = no pagination) */
	pagination?: PaginationState;

	/** Callback when page changes */
	onPageChange?: (page: number) => void;

	/** Callback when page size changes */
	onPageSizeChange?: (size: number) => void;

	/** Available page sizes */
	pageSizes?: number[];

	/** Enable selection for bulk operations */
	selectable?: boolean;

	/** Custom columns for table view */
	columns?: TableColumn[];

	/** Storage locations (for grid view storage dropdown) */
	storageLocations?: StorageLocation[];

	/** Callback when item is removed (e.g., deleted from inventory) */
	onItemRemove?: (itemId: string | number) => void;

	/** Empty state message */
	emptyMessage?: string;

	/** Empty state action slot */
	emptyAction?: Snippet;

	/** Header area slot (for stats, search form, etc.) */
	header?: Snippet;

	/** Filter controls slot */
	filters?: Snippet;

	/** Persist view preference to localStorage */
	persistViewPreference?: boolean;

	/** Local storage key for view preference */
	storageKey?: string;
}

/**
 * Context type for child components
 */
export interface CardCollectionContext {
	viewMode: ViewMode;
	selectable: boolean;
	storageLocations: StorageLocation[];
	onItemRemove?: (itemId: string | number) => void;
}
