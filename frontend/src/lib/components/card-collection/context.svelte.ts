/**
 * CardCollection context for sharing state with child components
 */

import { getContext, setContext } from 'svelte';
import type { CardCollectionContext, ViewMode, StorageLocation } from './types';

const CONTEXT_KEY = Symbol('card-collection');

/**
 * Set the CardCollection context (called by CardCollection.svelte)
 */
export function setCardCollectionContext(context: CardCollectionContext) {
	setContext(CONTEXT_KEY, context);
}

/**
 * Get the CardCollection context (called by child components)
 */
export function getCardCollectionContext(): CardCollectionContext {
	const context = getContext<CardCollectionContext | undefined>(CONTEXT_KEY);
	if (!context) {
		throw new Error('CardCollection context not found. Ensure component is used within CardCollection.');
	}
	return context;
}

/**
 * Try to get the CardCollection context, returning undefined if not available
 */
export function tryGetCardCollectionContext(): CardCollectionContext | undefined {
	return getContext<CardCollectionContext | undefined>(CONTEXT_KEY);
}

/**
 * Create a CardCollectionContext object
 */
export function createCardCollectionContext(options: {
	viewMode: ViewMode;
	selectable: boolean;
	storageLocations: StorageLocation[];
	onItemRemove?: (itemId: string | number) => void;
}): CardCollectionContext {
	return {
		viewMode: options.viewMode,
		selectable: options.selectable,
		storageLocations: options.storageLocations,
		onItemRemove: options.onItemRemove
	};
}
