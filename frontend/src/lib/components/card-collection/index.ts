// Main component
export { default as CardCollection } from './CardCollection.svelte';

// Types
export type {
	ViewMode,
	CardSourceType,
	DisplayableCard,
	PaginationState,
	TableColumn,
	CardCollectionProps,
	CardCollectionContext
} from './types';

// Adapters
export {
	fromInventoryCard,
	fromSearchResult,
	fromListItem,
	adaptInventoryCards,
	adaptSearchResults,
	adaptListItems,
	getSourceData,
	isInventoryCard,
	isSearchResult,
	isListItem,
	isFoilTreatment
} from './adapters';

// Context
export {
	setCardCollectionContext,
	getCardCollectionContext,
	tryGetCardCollectionContext,
	createCardCollectionContext
} from './context.svelte';

// Views (for advanced customization)
export { default as GridView } from './views/GridView.svelte';
export { default as TableView } from './views/TableView.svelte';

// Controls (for custom layouts)
export { default as ViewToggle } from './controls/ViewToggle.svelte';
export { default as PageSizeSelector } from './controls/PageSizeSelector.svelte';
export { default as CardFilter } from './controls/CardFilter.svelte';
