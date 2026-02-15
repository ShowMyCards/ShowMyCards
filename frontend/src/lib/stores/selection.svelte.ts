/**
 * Selection store for bulk operations
 *
 * Tracks selected inventory IDs for bulk move, delete, and other operations.
 *
 * @example
 * ```svelte
 * <script>
 *   import { selection } from '$lib';
 *
 *   function handleCheckbox(id: number) {
 *     selection.toggle(id);
 *   }
 * </script>
 *
 * <input type="checkbox" checked={selection.isSelected(id)} onchange={() => handleCheckbox(id)} />
 *
 * {#if selection.count > 0}
 *   <button onclick={() => handleBulkDelete()}>Delete {selection.count} items</button>
 * {/if}
 * ```
 */

class SelectionStore {
	/** Set of selected inventory IDs */
	private selectedIds = $state(new Set<number>());

	/** Number of selected items */
	get count(): number {
		return this.selectedIds.size;
	}

	/** Check if an ID is selected */
	isSelected(id: number): boolean {
		return this.selectedIds.has(id);
	}

	/** Select an ID */
	select(id: number) {
		this.selectedIds.add(id);
		// Trigger reactivity
		this.selectedIds = new Set(this.selectedIds);
	}

	/** Deselect an ID */
	deselect(id: number) {
		this.selectedIds.delete(id);
		// Trigger reactivity
		this.selectedIds = new Set(this.selectedIds);
	}

	/** Toggle selection of an ID */
	toggle(id: number) {
		if (this.selectedIds.has(id)) {
			this.deselect(id);
		} else {
			this.select(id);
		}
	}

	/** Select multiple IDs */
	selectMany(ids: number[]) {
		for (const id of ids) {
			this.selectedIds.add(id);
		}
		this.selectedIds = new Set(this.selectedIds);
	}

	/** Deselect multiple IDs */
	deselectMany(ids: number[]) {
		for (const id of ids) {
			this.selectedIds.delete(id);
		}
		this.selectedIds = new Set(this.selectedIds);
	}

	/** Clear all selections */
	clear() {
		this.selectedIds = new Set();
	}

	/** Get all selected IDs as an array */
	getSelected(): number[] {
		return Array.from(this.selectedIds);
	}

	/** Select all from a list of IDs */
	selectAll(ids: number[]) {
		this.selectedIds = new Set(ids);
	}
}

/**
 * Global selection store instance
 */
export const selection = new SelectionStore();
