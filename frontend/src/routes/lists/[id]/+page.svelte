<script lang="ts">
	import { browser } from '$app/environment';
	import { invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { deserialize, enhance } from '$app/forms';
	import type { PageData, ActionData } from './$types';
	import {
		PageHeader,
		ListStatsBar,
		ListCardGrid,
		ListCardTable,
		ListSearchResults,
		EmptyState,
		ViewToggle,
		CardFilter,
		getCardTreatmentName,
		sortCardsBySetAndCollector,
		groupCardsByTreatment,
		notifications,
		usePersistedViewMode,
		type ScryfallCard,
		type EnrichedListItem
	} from '$lib';
	import { ArrowLeft, Search } from '@lucide/svelte';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let searchQuery = $state('');
	let adding = $state(false);
	let searching = $state(false);

	// View mode state - persisted to localStorage
	const view = usePersistedViewMode('list-view-mode', 'table');

	// Client-side filtering state
	let filterText = $state('');
	let statusFilter = $state<'all' | 'collected' | 'uncollected'>('all');
	const PAGE_SIZE = 24;
	let currentPage = $state(1);

	function handleSearchChange(text: string) {
		filterText = text;
		currentPage = 1;
	}

	function handleStatusChange(status: 'all' | 'collected' | 'uncollected') {
		statusFilter = status;
		currentPage = 1;
	}

	function handlePageChange(page: number) {
		currentPage = page;
	}

	function filterItems(items: EnrichedListItem[]): EnrichedListItem[] {
		let filtered = items;

		if (statusFilter === 'collected') {
			filtered = filtered.filter((item) => item.collected_quantity >= item.desired_quantity);
		} else if (statusFilter === 'uncollected') {
			filtered = filtered.filter((item) => item.collected_quantity < item.desired_quantity);
		}

		if (filterText.trim()) {
			const search = filterText.toLowerCase().trim();
			filtered = filtered.filter((item) => {
				const name = (item.name || '').toLowerCase();
				const setName = (item.set_name || '').toLowerCase();
				const treatment = (item.treatment || '').toLowerCase();
				const treatmentName = getCardTreatmentName(
					[item.treatment],
					[],
					item.treatment
				).toLowerCase();

				return (
					name.includes(search) ||
					setName.includes(search) ||
					treatment.includes(search) ||
					treatmentName.includes(search)
				);
			});
		}

		return filtered;
	}

	// Filtered and paginated items
	const filteredItems = $derived(filterItems(data.items));
	const totalFilteredPages = $derived(Math.ceil(filteredItems.length / PAGE_SIZE) || 1);
	const paginatedItems = $derived(
		filteredItems.slice((currentPage - 1) * PAGE_SIZE, currentPage * PAGE_SIZE)
	);

	// Update search query when form data changes
	$effect(() => {
		if (form?.query) {
			searchQuery = form.query;
		}
	});

	// Display data load errors via notifications store (browser only)
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	// Get search results from form action response
	const searchResults = $derived<ScryfallCard[]>(form?.searchResults || []);

	function isInList(scryfallId: string, treatment: string): boolean {
		return data.items.some(
			(item) => item.scryfall_id === scryfallId && item.treatment === treatment
		);
	}

	// Use aggregate stats from backend (calculated across ALL items, not just current page)
	const stats = $derived({
		totalWanted: data.totalWanted || 0,
		totalCollected: data.totalCollected || 0,
		completion: data.completionPercent || 0,
		collectedValue: data.totalCollectedValue || 0,
		remainingValue: data.totalRemainingValue || 0
	});

	async function bulkAddTreatment(treatment: string | null) {
		if (adding) return;
		adding = true;

		const sorted = sortCardsBySetAndCollector(searchResults);
		const grouped = groupCardsByTreatment(sorted);

		const itemsToAdd: Array<{
			scryfall_id: string;
			oracle_id: string;
			treatment: string;
			desired_quantity: number;
		}> = [];

		if (treatment) {
			const cardsToAdd = grouped.cardsByTreatment.get(treatment) || [];
			for (const card of cardsToAdd) {
				if (!isInList(card.id, treatment)) {
					itemsToAdd.push({
						scryfall_id: card.id,
						oracle_id: card.oracle_id,
						treatment: treatment,
						desired_quantity: 1
					});
				}
			}
		} else {
			for (const card of sorted) {
				const treatments = card.finishes && card.finishes.length > 0 ? card.finishes : ['nonfoil'];
				for (const cardTreatment of treatments) {
					if (!isInList(card.id, cardTreatment)) {
						itemsToAdd.push({
							scryfall_id: card.id,
							oracle_id: card.oracle_id,
							treatment: cardTreatment,
							desired_quantity: 1
						});
					}
				}
			}
		}

		if (itemsToAdd.length === 0) {
			notifications.warning('All cards are already in the list!');
			adding = false;
			return;
		}

		const formData = new FormData();
		formData.append('items', JSON.stringify(itemsToAdd));

		try {
			const response = await fetch(`?/addItems`, {
				method: 'POST',
				body: formData
			});

			const result = deserialize(await response.text());
			if (result.type === 'success') {
				await invalidateAll();
			} else {
				notifications.error('Failed to add items. Please try again.');
			}
		} catch (error) {
			notifications.error('Error adding items: ' + error);
		} finally {
			adding = false;
		}
	}
</script>

<svelte:head>
	<title>{data.list?.name || 'List'} - ShowMyCards</title>
</svelte:head>

<div class="mx-auto px-4 py-8">
	{#if data.list}
		<PageHeader title={data.list.name} description={data.list.description}>
			{#snippet actions()}
				<a href={resolve('/lists')} class="btn bg-base-100 btn-sm">
					<ArrowLeft class="w-4 h-4" />
					Back
				</a>
			{/snippet}
		</PageHeader>

		<!-- Summary Stats -->
		<ListStatsBar
			totalWanted={stats.totalWanted}
			totalCollected={stats.totalCollected}
			collectedValue={stats.collectedValue}
			remainingValue={stats.remainingValue}
			completion={stats.completion} />

		<!-- Tabs for the List -->
		<div class="tabs tabs-boxed mb-6">
			<input type="radio" name="list_tabs" class="tab" aria-label="Cards" checked={true} />
			<div class="tab-content bg-base-100 border-base-300 rounded-box p-6">
				<!-- Filter bar -->
				<div class="flex flex-col gap-4 mb-4">
					<div class="flex flex-col sm:flex-row gap-4 items-start sm:items-center justify-between">
						<CardFilter
							searchText={filterText}
							onSearchChange={handleSearchChange}
							{statusFilter}
							onStatusChange={handleStatusChange}
							showStatusFilter={true}
							placeholder="Filter by name, set, or treatment..." />
						<ViewToggle viewMode={view.viewMode} onViewModeChange={view.setViewMode} />
					</div>
					<div class="text-sm opacity-70">
						{#if filterText || statusFilter !== 'all'}
							Showing {filteredItems.length} of {data.items.length} cards
							{#if totalFilteredPages > 1}
								(page {currentPage} of {totalFilteredPages})
							{/if}
						{:else}
							{data.items.length}
							{data.items.length === 1 ? 'card' : 'cards'}
							{#if totalFilteredPages > 1}
								(page {currentPage} of {totalFilteredPages})
							{/if}
						{/if}
					</div>
				</div>

				{#if data.items.length === 0}
					<EmptyState message="No cards in this list yet">
						<p class="text-sm opacity-70">Use the "Manage List" tab to add cards</p>
					</EmptyState>
				{:else if filteredItems.length === 0}
					<EmptyState message="No cards match your filter">
						<p class="text-sm opacity-70">Try adjusting your search or filter criteria</p>
					</EmptyState>
				{:else if view.viewMode === 'grid'}
					<ListCardGrid
						items={paginatedItems}
						{currentPage}
						totalPages={totalFilteredPages}
						onPageChange={handlePageChange} />
				{:else}
					<ListCardTable
						items={paginatedItems}
						{currentPage}
						totalPages={totalFilteredPages}
						onPageChange={handlePageChange} />
				{/if}
			</div>

			<!-- Add Cards to List -->
			<input type="radio" name="list_tabs" class="tab" aria-label="Manage List" />
			<div class="tab-content bg-base-100 border-base-300 rounded-box p-6">
				<h2 class="card-title">Add Cards to List</h2>

				<form
					method="POST"
					action="?/search"
					use:enhance={() => {
						searching = true;
						return async ({ update }) => {
							await update();
							searching = false;
						};
					}}
					class="flex gap-2">
					<input
						type="text"
						name="q"
						bind:value={searchQuery}
						placeholder="Search for cards..."
						class="input input-bordered flex-1"
						required />
					<button type="submit" disabled={searching} class="btn btn-primary">
						<Search class="w-4 h-4" />
						{searching ? 'Searching...' : 'Search'}
					</button>
				</form>

				<ListSearchResults
					{searchResults}
					{searchQuery}
					{adding}
					{isInList}
					onBulkAddTreatment={bulkAddTreatment} />
			</div>
		</div>
	{:else}
		<EmptyState message="List not found">
			<a href={resolve('/lists')} class="btn btn-primary">Back to Lists</a>
		</EmptyState>
	{/if}
</div>
