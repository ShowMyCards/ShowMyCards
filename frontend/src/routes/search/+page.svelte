<script lang="ts">
	import { browser } from '$app/environment';
	import { enhance, deserialize } from '$app/forms';
	import { resolve } from '$app/paths';
	import type { ActionData, PageData } from './$types';
	import {
		CardResultCard,
		PageHeader,
		StatsCard,
		EmptyState,
		Pagination,
		ViewToggle,
		CardFilter,
		TreatmentBadge,
		isFoilTreatment,
		getCardTreatmentName,
		notifications,
		usePersistedViewMode,
		type EnhancedCardResult
	} from '$lib';
	import { Search, Lightbulb } from '@lucide/svelte';
	import SetIcon from '$lib/components/SetIcon.svelte';

	let { form, data }: { form: ActionData; data: PageData } = $props();

	let searching = $state(false);
	let removedCardIds = $state(new Set<string>());
	let cards = $state<EnhancedCardResult[]>([]);
	let suggestions = $state<string[]>([]);
	let inputRef = $state<HTMLInputElement | null>(null);

	// View mode state with localStorage persistence
	const view = usePersistedViewMode('smc-search-view-mode', 'grid');

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

	// Sync cards with form data when search results arrive
	$effect(() => {
		if (form?.success && form.data?.data) {
			cards = form.data.data;
			currentPage = 1;
		}
	});

	// Fetch suggestions when search returns no results
	$effect(() => {
		if (!browser) return;
		if (form?.success && form.data?.data?.length === 0 && form.query) {
			fetchSuggestions(form.query);
		} else {
			suggestions = [];
		}
	});

	async function fetchSuggestions(query: string) {
		try {
			const formData = new FormData();
			formData.append('q', query);

			const response = await fetch('?/autocomplete', {
				method: 'POST',
				body: formData
			});

			const result = deserialize(await response.text());
			if (result.type === 'success' && result.data) {
				suggestions = (result.data as { suggestions: string[] }).suggestions || [];
			}
		} catch {
			suggestions = [];
		}
	}

	function searchSuggestion(suggestion: string) {
		if (inputRef) {
			inputRef.value = suggestion;
			inputRef.form?.requestSubmit();
		}
	}

	// Display search errors if present (browser only)
	let hasShownSearchError = $state(false);
	$effect(() => {
		if (!browser || hasShownSearchError) return;
		if (form?.error) {
			hasShownSearchError = true;
			notifications.error(form.error);
		}
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

	/**
	 * Get the primary treatment for display in table view
	 */
	function getPrimaryTreatment(card: EnhancedCardResult): string {
		return card.finishes[0] || 'nonfoil';
	}
</script>

<svelte:head>
	<title>Search Cards - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader
		title="Search Cards"
		description="Search Scryfall's database for Magic: The Gathering cards" />

	<div class="card bg-base-200 shadow-lg mb-6">
		<div class="card-body">
			<form
				method="POST"
				action="?/search"
				use:enhance={() => {
					searching = true;
					return async ({ update }) => {
						await update();
						searching = false;
					};
				}}>
				<div class="flex gap-2">
					<input
						bind:this={inputRef}
						type="text"
						name="q"
						value={form?.query || ''}
						placeholder="Search for cards..."
						class="input input-bordered flex-1"
						required />
					<button type="submit" disabled={searching} class="btn btn-primary">
						{#if searching}
							<span class="loading loading-spinner loading-sm"></span>
							Searching...
						{:else}
							<Search class="w-4 h-4" />
							Search
						{/if}
					</button>
				</div>
			</form>
		</div>
	</div>

	{#if form?.success && form.data}
		<StatsCard
			stats={[
				{
					title: 'Results Found',
					value: form.data.total_cards || 0,
					description: form.data.has_more ? 'Showing first page' : 'All results shown'
				},
				{
					title: 'Displaying',
					value: filteredCards.length,
					description: filterText ? 'Matching filter' : 'Cards on this page'
				}
			]}
			class="mb-6 w-full" />

		<!-- Filter bar -->
		<div class="flex flex-col gap-4 mb-4">
			<div class="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
				<CardFilter
					searchText={filterText}
					onSearchChange={handleSearchChange}
					showStatusFilter={false}
					placeholder="Filter results by name, set, or treatment..." />
				<ViewToggle viewMode={view.viewMode} onViewModeChange={view.setViewMode} />
			</div>
			{#if filterText || totalFilteredPages > 1}
				<div class="text-sm opacity-70">
					{#if filterText}
						Showing {filteredCards.length} of {cards.length} cards
					{/if}
					{#if totalFilteredPages > 1}
						(page {currentPage} of {totalFilteredPages})
					{/if}
				</div>
			{/if}
		</div>

		{#if cards.length > 0 && filteredCards.length === 0}
			<EmptyState message="No cards match your filter">
				<p class="text-sm opacity-70">Try adjusting your search criteria</p>
			</EmptyState>
		{:else if paginatedCards.length > 0}
			{#if view.viewMode === 'grid'}
				<!-- Grid View -->
				<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
					{#each paginatedCards as card (card.id)}
						<CardResultCard
							{card}
							onremove={handleRemove}
							storageLocations={data.storageLocations} />
					{/each}
				</div>
			{:else}
				<!-- Table View -->
				<div class="overflow-x-auto">
					<table class="table table-zebra">
						<thead>
							<tr>
								<th>Card Name</th>
								<th>Set</th>
								<th>#</th>
								<th>Treatment(s)</th>
								<th>In Collection</th>
							</tr>
						</thead>
						<tbody>
							{#each paginatedCards as card (card.id)}
								{@const primaryTreatment = getPrimaryTreatment(card)}
								{@const isFoil = isFoilTreatment(primaryTreatment)}
								{@const totalQty = card.inventory.total_quantity}
								<tr class="hover:bg-base-300">
									<td>
										<a href={resolve(`/cards/${card.id}`)} class="font-semibold hover:text-primary">
											{card.name}
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
										{#if totalQty > 0}
											<span class="badge badge-primary">{totalQty}</span>
										{:else}
											<span class="opacity-50">-</span>
										{/if}
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
					<Pagination
						{currentPage}
						totalPages={totalFilteredPages}
						onPageChange={handlePageChange} />
				</div>
			{/if}
		{:else}
			<EmptyState message="No cards found" />

			{#if suggestions.length > 0}
				<div class="mt-4 p-4 bg-base-200 rounded-lg">
					<div class="flex items-center gap-2 text-sm opacity-70 mb-3">
						<Lightbulb class="w-4 h-4" />
						<span>Did you mean:</span>
					</div>
					<div class="flex flex-wrap gap-2">
						{#each suggestions as suggestion (suggestion)}
							<button
								type="button"
								class="btn btn-sm btn-outline"
								onclick={() => searchSuggestion(suggestion)}>
								{suggestion}
							</button>
						{/each}
					</div>
				</div>
			{/if}
		{/if}
	{/if}
</div>
