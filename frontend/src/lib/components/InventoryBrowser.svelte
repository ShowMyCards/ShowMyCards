<script lang="ts">
	import { browser } from '$app/environment';
	import { invalidateAll } from '$app/navigation';
	import {
		CardResultCard,
		EmptyState,
		Pagination,
		BulkActionsBar,
		ViewToggle,
		CardFilter,
		TreatmentBadge,
		isFoilTreatment,
		getCardTreatmentName,
		notifications,
		selection,
		usePersistedViewMode,
		type EnhancedCardResult,
		type StorageLocationWithCount
	} from '$lib';
	import SetIcon from '$lib/components/SetIcon.svelte';
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
				const name = (card.name || '').toLowerCase();
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
			<a href="/search" class="btn btn-primary">Search for Cards</a>
		</EmptyState>
	{:else if filteredCards.length === 0}
		<EmptyState message="No cards match your filter">
			<p class="text-sm opacity-70">Try adjusting your search criteria</p>
		</EmptyState>
	{:else if view.viewMode === 'grid'}
		<!-- Grid View -->
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4 pb-20">
			{#each paginatedCards as card (card.id)}
				<CardResultCard {card} onremove={handleRemove} selectable />
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
						<th>Treatment(s)</th>
						<th>Qty</th>
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
								<a href="/cards/{card.id}" class="font-semibold hover:text-primary">
									{card.name}
								</a>
							</td>
							<td>
								<div class="flex items-center gap-2">
									<SetIcon setCode={card.set_code} setName={card.set_name} rarity="common" {isFoil} />
									<span class="text-sm">{card.set_name}</span>
								</div>
							</td>
							<td>#{card.collector_number || '?'}</td>
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
</div>
