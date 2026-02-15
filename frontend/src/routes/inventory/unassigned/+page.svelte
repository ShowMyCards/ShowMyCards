<script lang="ts">
	import type { PageData } from './$types';
	import { PageHeader, InventoryBrowser } from '$lib';
	import { ArrowLeft } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();
</script>

<svelte:head>
	<title>Unassigned Cards - Inventory Browser - ShowMyCards</title>
</svelte:head>

<InventoryBrowser
	cards={data.cards}
	allLocations={data.allLocations}
	error={data.error}
	emptyMessage="No unassigned cards">
	{#snippet header()}
		<PageHeader
			title="Unassigned Cards"
			description={`Cards without a storage location • ${data.totalCards} card${data.totalCards === 1 ? '' : 's'}`}>
			{#snippet actions()}
				<a href="/inventory" class="btn bg-base-100 btn-sm">
					<ArrowLeft class="w-4 h-4" />
					Back to Locations
				</a>
			{/snippet}
		</PageHeader>
	{/snippet}
</InventoryBrowser>
