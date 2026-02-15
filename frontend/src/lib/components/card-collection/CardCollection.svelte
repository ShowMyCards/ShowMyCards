<script lang="ts">
	import { browser } from '$app/environment';
	import type { Snippet } from 'svelte';
	import { EmptyState, Pagination } from '$lib';
	import type { DisplayableCard, ViewMode, PaginationState, TableColumn, StorageLocation } from './types';
	import GridView from './views/GridView.svelte';
	import TableView from './views/TableView.svelte';
	import ViewToggle from './controls/ViewToggle.svelte';
	import PageSizeSelector from './controls/PageSizeSelector.svelte';

	interface Props {
		/** Items to display */
		items: DisplayableCard[];

		/** Loading state */
		loading?: boolean;

		/** Default view mode */
		viewMode?: ViewMode;

		/** Callback when view mode changes */
		onViewModeChange?: (mode: ViewMode) => void;

		/** Pagination state */
		pagination?: PaginationState;

		/** Callback when page changes */
		onPageChange?: (page: number) => void;

		/** Available page sizes */
		pageSizes?: number[];

		/** Callback when page size changes */
		onPageSizeChange?: (size: number) => void;

		/** Enable selection for bulk operations */
		selectable?: boolean;

		/** Custom columns for table view */
		columns?: TableColumn[];

		/** Storage locations for grid view */
		storageLocations?: StorageLocation[];

		/** Callback when item is removed */
		onItemRemove?: (itemId: string | number) => void;

		/** Empty state message */
		emptyMessage?: string;

		/** Empty state action slot */
		emptyAction?: Snippet;

		/** Header slot */
		header?: Snippet;

		/** Filters slot */
		filters?: Snippet;

		/** Persist view preference to localStorage */
		persistViewPreference?: boolean;

		/** localStorage key for view preference */
		storageKey?: string;
	}

	let {
		items,
		loading = false,
		viewMode: initialViewMode = 'grid',
		onViewModeChange,
		pagination,
		onPageChange,
		pageSizes = [20, 50, 100],
		onPageSizeChange,
		selectable = false,
		columns,
		storageLocations = [],
		onItemRemove,
		emptyMessage = 'No cards to display',
		emptyAction,
		header,
		filters,
		persistViewPreference = false,
		storageKey = 'default'
	}: Props = $props();

	const STORAGE_KEY_PREFIX = 'card-collection-view-';

	// Load persisted view mode or use initial
	function getStoredViewMode(): ViewMode {
		if (!browser || !persistViewPreference) return initialViewMode;
		const stored = localStorage.getItem(`${STORAGE_KEY_PREFIX}${storageKey}`);
		return stored === 'table' ? 'table' : stored === 'grid' ? 'grid' : initialViewMode;
	}

	// Internal view mode state
	let currentViewMode = $state<ViewMode>(getStoredViewMode());

	// Persist view mode when it changes
	$effect(() => {
		if (browser && persistViewPreference) {
			localStorage.setItem(`${STORAGE_KEY_PREFIX}${storageKey}`, currentViewMode);
		}
	});

	function handleViewModeChange(mode: ViewMode) {
		currentViewMode = mode;
		onViewModeChange?.(mode);
	}

	function handlePageChange(page: number) {
		onPageChange?.(page);
	}

	function handlePageSizeChange(size: number) {
		onPageSizeChange?.(size);
	}

	function handleItemRemove(itemId: string | number) {
		onItemRemove?.(itemId);
	}
</script>

<div class="card-collection">
	<!-- Header slot -->
	{#if header}
		<div class="mb-4">
			{@render header()}
		</div>
	{/if}

	<!-- Controls bar -->
	<div class="flex flex-wrap items-center justify-between gap-4 mb-4">
		<!-- Filters slot (left side) -->
		{#if filters}
			<div class="flex items-center gap-4">
				{@render filters()}
			</div>
		{:else}
			<div></div>
		{/if}

		<!-- View controls (right side) -->
		<div class="flex items-center gap-4">
			{#if pagination && onPageSizeChange}
				<PageSizeSelector
					pageSize={pagination.pageSize}
					{pageSizes}
					onPageSizeChange={handlePageSizeChange} />
			{/if}
			<ViewToggle viewMode={currentViewMode} onViewModeChange={handleViewModeChange} />
		</div>
	</div>

	<!-- Loading state -->
	{#if loading}
		<div class="flex justify-center py-12">
			<span class="loading loading-spinner loading-lg"></span>
		</div>
	{:else if items.length === 0}
		<!-- Empty state -->
		<EmptyState message={emptyMessage}>
			{#if emptyAction}
				{@render emptyAction()}
			{/if}
		</EmptyState>
	{:else}
		<!-- View content -->
		<div class="pb-20">
			{#if currentViewMode === 'grid'}
				<GridView {items} {storageLocations} {selectable} onItemRemove={handleItemRemove} />
			{:else}
				<TableView {items} {columns} {selectable} onItemRemove={handleItemRemove} />
			{/if}
		</div>

		<!-- Pagination -->
		{#if pagination && pagination.totalPages > 1}
			<div class="mt-6">
				<Pagination
					currentPage={pagination.currentPage}
					totalPages={pagination.totalPages}
					onPageChange={handlePageChange} />
			</div>
		{/if}
	{/if}
</div>
