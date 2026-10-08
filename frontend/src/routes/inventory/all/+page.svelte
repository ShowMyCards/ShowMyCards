<script lang="ts">
	import type { PageData } from './$types';
	import { PageHeader, InventoryBrowser } from '$lib';
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>All Cards - Inventory Browser - ShowMyCards</title>
</svelte:head>

<InventoryBrowser
	cards={data.cards}
	allLocations={[]}
	error={data.error}
	emptyMessage="No cards in your collection"
	readOnly
	showPriceSort
	showLocations>
	{#snippet header()}
		<PageHeader
			title="All Cards"
			description={data.error
				? 'Cards across all storage locations, including unassigned cards'
				: `All locations, including unassigned • ${data.cards.length} ${data.cards.length === 1 ? 'printing' : 'printings'} • ${data.totalCopies} ${data.totalCopies === 1 ? 'copy' : 'copies'}`}>
			{#snippet actions()}
				<a href={resolve('/inventory')} class="btn bg-base-100 btn-sm">
					<ArrowLeft class="w-4 h-4" />
					Back to Locations
				</a>
			{/snippet}
		</PageHeader>
	{/snippet}
</InventoryBrowser>
