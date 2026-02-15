<script lang="ts">
	import type { PageData } from './$types';
	import { CardResultCard, PageHeader, EmptyState, StatsCard } from '$lib';
	import { resolve } from '$app/paths';

	let { data }: { data: PageData } = $props();

	let removedCardIds = $state(new Set<string>());

	const visibleCards = $derived(data.cards.filter((c) => !removedCardIds.has(c.id)));

	function handleRemove(cardId: string) {
		removedCardIds.add(cardId);
		removedCardIds = removedCardIds;
	}
</script>

<svelte:head>
	<title>Recently Added - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader
		title="Recently Added"
		description="Your most recently added cards - quick access for quantity adjustments">
		{#snippet actions()}
			<a href={resolve('/inventory')} class="btn btn-ghost">
				View All Inventory
			</a>
		{/snippet}
	</PageHeader>

	<StatsCard
		stats={[
			{
				title: 'Recent Cards',
				value: visibleCards.length,
				description: 'Last 20 unique cards added'
			},
			{
				title: 'Total in Collection',
				value: data.total,
				description: 'Across all storage locations'
			}
		]}
		class="mb-6 w-full" />

	{#if visibleCards.length > 0}
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
			{#each visibleCards as card (card.id)}
				<CardResultCard {card} onremove={handleRemove} storageLocations={data.storageLocations} />
			{/each}
		</div>
	{:else}
		<EmptyState message="No cards in your collection yet. Search for cards and add them to your inventory.">
			<a href={resolve('/search')} class="btn btn-primary">Search Cards</a>
		</EmptyState>
	{/if}
</div>
