<script lang="ts">
	import type { PageData } from './$types';
	import { PageHeader, InventoryBrowser } from '$lib';
	import { resolve } from '$app/paths';
	import { ArrowLeft } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>{data.location?.name || 'Location'} - Inventory Browser - ShowMyCards</title>
</svelte:head>

<InventoryBrowser
	cards={data.cards}
	allLocations={data.allLocations}
	error={data.error}
	emptyMessage="No cards in this location">
	{#snippet header()}
		{#if data.location}
			<PageHeader
				title={data.location.name}
				description={`${data.location.storage_type} • ${data.totalCards} card${data.totalCards === 1 ? '' : 's'}`}>
				{#snippet actions()}
					<a href={resolve('/inventory')} class="btn bg-base-100 btn-sm">
						<ArrowLeft class="w-4 h-4" />
						Back to Locations
					</a>
				{/snippet}
			</PageHeader>
		{/if}
	{/snippet}
</InventoryBrowser>
