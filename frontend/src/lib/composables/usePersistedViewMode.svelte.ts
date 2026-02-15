import { browser } from '$app/environment';
import type { ViewMode } from '$lib';

/**
 * Reactive view mode state persisted to localStorage.
 *
 * Access `viewMode` as a property — do NOT destructure, as that breaks reactivity.
 *
 * @example
 * ```ts
 * const view = usePersistedViewMode('smc-search-view-mode', 'grid');
 * // In template: {#if view.viewMode === 'grid'}
 * // To change:   view.setViewMode('table')
 * ```
 */
export class PersistedViewMode {
	viewMode = $state<ViewMode>('grid');
	#storageKey: string;

	constructor(storageKey: string, defaultMode: ViewMode) {
		this.viewMode = defaultMode;
		this.#storageKey = storageKey;

		// Initialize from localStorage
		$effect(() => {
			if (browser) {
				const saved = localStorage.getItem(this.#storageKey);
				if (saved === 'grid' || saved === 'table') {
					this.viewMode = saved;
				}
			}
		});

		// Persist changes
		$effect(() => {
			if (browser) {
				localStorage.setItem(this.#storageKey, this.viewMode);
			}
		});
	}

	setViewMode = (mode: ViewMode) => {
		this.viewMode = mode;
	};
}

/**
 * Creates a persisted view mode preference backed by localStorage.
 *
 * @param storageKey - The localStorage key to persist the view mode under
 * @param defaultMode - The default view mode if no saved value exists (defaults to 'grid')
 *
 * @example
 * ```ts
 * const view = usePersistedViewMode('smc-search-view-mode', 'grid');
 * // In template: {#if view.viewMode === 'grid'}
 * // To change:   view.setViewMode('table')
 * ```
 */
export function usePersistedViewMode(storageKey: string, defaultMode: ViewMode = 'grid') {
	return new PersistedViewMode(storageKey, defaultMode);
}
