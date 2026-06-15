<script lang="ts">
	import { browser } from '$app/environment';
	import { invalidateAll } from '$app/navigation';
	import {
		CardResultCard,
		EmptyState,
		Pagination,
		BulkActionsBar,
		SplitMoveModal,
		ViewToggle,
		CardFilter,
		TreatmentBadge,
		isFoilTreatment,
		getCardTreatmentName,
		getDisplayName,
		notifications,
		selection,
		usePersistedViewMode,
		type EnhancedCardResult,
		type Inventory,
		type StorageLocationWithCount
	} from '$lib';
	import SetIcon from '$lib/components/SetIcon.svelte';
	import { FolderInput } from '@lucide/svelte';
	import { resolve } from '$app/paths';
	import type { Snippet } from 'svelte';

	interface Props {
		cards: EnhancedCardResult[];
		allLocations: StorageLocationWithCount[];
		error?: string;
		emptyMessage?: string;
		header: Snippet;
	}

	let {
		cards,
		allLocations,
		error: loadError,
		emptyMessage = 'No cards found',
		header
	}: Props = $props();

	let removedCardIds = $state(new Set<string>());

	// The inventory stack currently targeted by the split-move dialog. A stack may span
	// multiple rows (the add flow never merges), so we track the aggregated quantity.
	let moveTarget = $state<{ inv: Inventory; available: number } | null>(null);

	/**
	 * Group a card's inventory rows into user-facing stacks by treatment. Within a single
	 * inventory view all rows share one location, so (treatment) identifies a stack.
	 */
	function treatmentGroups(items: Inventory[]): { rep: Inventory; total: number }[] {
		const groups: { rep: Inventory; total: number }[] = [];
		for (const inv of items) {
			const existing = groups.find((g) => g.rep.treatment === inv.treatment);
			if (existing) {
				existing.total += inv.quantity;
			} else {
				groups.push({ rep: inv, total: inv.quantity });
			}
		}
		return groups;
	}

	// View mode state with localStorage persistence
	const view = usePersistedViewMode('smc-inventory-view-mode', 'grid');

	// Client-side filtering state
	let filterText = $state('');
	const PAGE_SIZE = 24;
	let currentPage = $state(1);

	function handleSearchChange(text: string) {
		filterText = text;
		currentPage = 1;
	}

	function handlePageChange(page: number) {
		currentPage = page;
	}

	// Display load error if present (browser only)
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (loadError) {
			hasShownLoadError = true;
			notifications.error(loadError);
		}
	});

	// Clear selection when leaving the page
	$effect(() => {
		return () => {
			selection.clear();
		};
	});

	/**
	 * Filter cards by search text (name, set, treatment)
	 */
	function filterCards(cardsList: EnhancedCardResult[]): EnhancedCardResult[] {
		let filtered = cardsList.filter((c) => !removedCardIds.has(c.id));

		if (filterText.trim()) {
			const search = filterText.toLowerCase().trim();
			filtered = filtered.filter((card) => {
				const name = `${card.name ?? ''} ${card.printed_name ?? ''}`.toLowerCase();
				const setName = (card.set_name || '').toLowerCase();
				const treatmentName = getCardTreatmentName(
					card.finishes,
					card.frame_effects ?? [],
					card.finishes[0] || 'nonfoil',
					card.promo_types ?? []
				).toLowerCase();

				return name.includes(search) || setName.includes(search) || treatmentName.includes(search);
			});
		}

		return filtered;
	}

	// Filtered and paginated cards
	const filteredCards = $derived(filterCards(cards));
	const totalFilteredPages = $derived(Math.ceil(filteredCards.length / PAGE_SIZE) || 1);
	const paginatedCards = $derived(
		filteredCards.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
	);

	function handleRemove(cardId: string) {
		removedCardIds.add(cardId);
		removedCardIds = removedCardIds; // Trigger reactivity
	}

	function handleBulkComplete() {
		invalidateAll();
	}

	/**
	 * Get the primary treatment for display in table view
	 */
	function getPrimaryTreatment(card: EnhancedCardResult): string {
		return card.finishes[0] || 'nonfoil';
	}
</script>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	{@render header()}

	<!-- Filter bar -->
	<div class="flex flex-col gap-4 mb-4">
		<div class="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
			<CardFilter
				searchText={filterText}
				onSearchChange={handleSearchChange}
				showStatusFilter={false}
				placeholder="Filter by name, set, or treatment..." />
			<ViewToggle viewMode={view.viewMode} onViewModeChange={view.setViewMode} />
		</div>
		<div class="text-sm opacity-70">
			{#if filterText}
				Showing {filteredCards.length} of {cards.length} cards
			{:else}
				{cards.length} {cards.length === 1 ? 'card' : 'cards'}
			{/if}
			{#if totalFilteredPages > 1}
				(page {currentPage} of {totalFilteredPages})
			{/if}
		</div>
	</div>

	{#if cards.length === 0}
		<EmptyState message={emptyMessage}>
			<a href={resolve('/search')} class="btn btn-primary">Search for Cards</a>
		</EmptyState>
	{:else if filteredCards.length === 0}
		<EmptyState message="No cards match your filter">
			<p class="text-sm opacity-70">Try adjusting your search criteria</p>
		</EmptyState>
	{:else if view.viewMode === 'grid'}
		<!-- Grid View -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-20">
			<!--
				Key intentionally includes total_quantity, not just card.id. CardResultCard
				captures card.inventory into local $state once on mount and never re-syncs from
				the prop (it diverges via optimistic +/-). After a split-move calls
				invalidateAll(), the reloaded card keeps the same id but a new quantity; keying
				on id alone would reuse the stale instance and show the pre-move quantity. The
				quantity in the key forces a remount so the fresh data is read. Do not simplify
				to (card.id) without first making CardResultCard reconcile from props.
			-->
			{#each paginatedCards as card (`${card.id}-${card.inventory.total_quantity}`)}
				<CardResultCard
					{card}
					onremove={handleRemove}
					selectable
					onSplitMove={(inv, available) => (moveTarget = { inv, available })} />
			{/each}
		</div>
	{:else}
		<!-- Table View -->
		<div class="overflow-x-auto pb-20">
			<table class="table table-zebra">
				<thead>
					<tr>
						<th>
							<input
								type="checkbox"
								class="checkbox checkbox-accent checkbox-sm"
								aria-label="Select all cards on this page"
								checked={paginatedCards.every((c) =>
									c.inventory.this_printing.every((inv) => selection.isSelected(inv.id))
								)}
								onchange={(e) => {
									const allIds = paginatedCards.flatMap((c) =>
										c.inventory.this_printing.map((inv) => inv.id)
									);
									if (e.currentTarget.checked) {
										selection.selectMany(allIds);
									} else {
										selection.deselectMany(allIds);
									}
								}} />
						</th>
						<th>Card Name</th>
						<th>Set</th>
						<th>#</th>
						<th>Language</th>
						<th>Treatment(s)</th>
						<th>Qty</th>
						<th>Actions</th>
					</tr>
				</thead>
				<tbody>
					{#each paginatedCards as card (card.id)}
						{@const primaryTreatment = getPrimaryTreatment(card)}
						{@const isFoil = isFoilTreatment(primaryTreatment)}
						{@const totalQty = card.inventory.total_quantity}
						{@const inventoryIds = card.inventory.this_printing.map((inv) => inv.id)}
						{@const isSelected = inventoryIds.every((id) => selection.isSelected(id))}
						<tr class="hover:bg-base-300">
							<td>
								<input
									type="checkbox"
									class="checkbox checkbox-accent checkbox-sm"
									checked={isSelected}
									onchange={() => {
										if (isSelected) {
											selection.deselectMany(inventoryIds);
										} else {
											selection.selectMany(inventoryIds);
										}
									}} />
							</td>
							<td>
								<a href={resolve(`/cards/${card.id}`)} class="font-semibold hover:text-primary">
									{getDisplayName(card)}
								</a>
							</td>
							<td>
								<div class="flex items-center gap-2">
									<SetIcon
										setCode={card.set_code}
										setName={card.set_name}
										rarity="common"
										{isFoil} />
									<span class="text-sm">{card.set_name}</span>
								</div>
							</td>
							<td>#{card.collector_number || '?'}</td>
							<td class="text-sm uppercase">{card.language}</td>
							<td>
								<div class="flex flex-wrap gap-1">
									{#each card.finishes as finish (finish)}
										<TreatmentBadge
											treatment={finish}
											finishes={card.finishes}
											frameEffects={card.frame_effects ?? []}
											promoTypes={card.promo_types ?? []}
											size="xs" />
									{/each}
								</div>
							</td>
							<td>
								<span class="badge badge-primary">{totalQty}</span>
							</td>
							<td>
								<div class="flex flex-wrap gap-1">
									{#each treatmentGroups(card.inventory.this_printing) as group (group.rep.id)}
										<button
											class="btn btn-ghost btn-xs"
											onclick={() => (moveTarget = { inv: group.rep, available: group.total })}
											title="Move copies to another location"
											aria-label="Move copies to another location">
											<FolderInput class="h-4 w-4" />
											Move
										</button>
									{/each}
								</div>
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	{/if}

	<!-- Pagination -->
	{#if totalFilteredPages > 1}
		<div class="mt-6">
			<Pagination {currentPage} totalPages={totalFilteredPages} onPageChange={handlePageChange} />
		</div>
	{/if}

	<BulkActionsBar locations={allLocations} onComplete={handleBulkComplete} />

	<SplitMoveModal
		open={!!moveTarget}
		inventory={moveTarget?.inv ?? null}
		availableQuantity={moveTarget?.available ?? 0}
		currentLocationId={moveTarget?.inv.storage_location_id}
		locations={allLocations}
		onClose={() => (moveTarget = null)}
		onComplete={handleBulkComplete} />
</div>
