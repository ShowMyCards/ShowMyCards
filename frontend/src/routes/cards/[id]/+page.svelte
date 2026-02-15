<script lang="ts">
	import type { PageData } from './$types';
	import { PageHeader, getCardTreatmentName } from '$lib';
	import { ArrowLeft, ExternalLink } from '@lucide/svelte';
	import PriceLozenge from '$lib/components/PriceLozenge.svelte';

	let { data }: { data: PageData } = $props();

	const card = $derived(data.card);
	const otherPrintings = $derived(data.otherPrintings);

	// Group inventory by treatment
	const inventoryByTreatment = $derived.by(() => {
		const map = new Map<string, number>();
		for (const inv of card.inventory.this_printing) {
			const treatment = inv.treatment || 'nonfoil';
			map.set(treatment, (map.get(treatment) || 0) + inv.quantity);
		}
		return map;
	});

	// Get available treatments
	const availableTreatments = $derived(card.finishes.length > 0 ? card.finishes : ['nonfoil']);
</script>

<svelte:head>
	<title>{card.name} - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-6xl">
	<!-- Back button -->
	<a href="/search" class="btn btn-ghost btn-sm mb-4">
		<ArrowLeft class="w-4 h-4" />
		Back to Search
	</a>

	<div class="grid grid-cols-1 lg:grid-cols-3 gap-8">
		<!-- Card Image -->
		<div class="lg:col-span-1">
			{#if card.image_uri}
				<figure class="rounded-lg overflow-hidden shadow-xl">
					<img src={card.image_uri} alt={card.name} class="w-full" />
				</figure>
			{:else}
				<div class="w-full aspect-5/7 bg-base-300 rounded-lg flex items-center justify-center">
					<p class="opacity-50">No image available</p>
				</div>
			{/if}

			<!-- External links -->
			<div class="mt-4 flex gap-2">
				<a
					href="https://scryfall.com/card/{card.id}"
					target="_blank"
					rel="noopener noreferrer"
					class="btn btn-outline btn-sm flex-1">
					<ExternalLink class="w-4 h-4" />
					Scryfall
				</a>
			</div>
		</div>

		<!-- Card Details -->
		<div class="lg:col-span-2">
			<PageHeader title={card.name} description={card.set_name || ''} />

			<!-- Prices & Inventory -->
			<div class="card bg-base-200 shadow-lg mb-6">
				<div class="card-body">
					<h3 class="card-title text-lg">Your Collection</h3>

					{#if card.inventory.total_quantity > 0}
						<div class="flex flex-wrap gap-4">
							{#each availableTreatments as treatment (treatment)}
								{@const quantity = inventoryByTreatment.get(treatment) || 0}
								{@const treatmentName = getCardTreatmentName(
									card.finishes,
									card.frame_effects || [],
									treatment
								)}
								{@const isFoil = treatment.includes('foil')}
								<div
									class="stat bg-base-100 rounded-lg p-4"
									class:bg-gradient-to-br={isFoil}
									class:from-yellow-100={isFoil}
									class:to-amber-200={isFoil}>
									<div class="stat-title">{treatmentName}</div>
									<div class="stat-value text-2xl">{quantity}</div>
									{#if card.prices}
										<div class="stat-desc">
											<PriceLozenge {treatment} prices={card.prices} />
										</div>
									{/if}
								</div>
							{/each}
						</div>

						<!-- Storage locations -->
						{#if card.inventory.this_printing.some((i) => i.storage_location)}
							<div class="mt-4">
								<h4 class="font-semibold mb-2">Storage Locations</h4>
								<div class="flex flex-wrap gap-2">
									{#each card.inventory.this_printing as inv (inv.id)}
										{#if inv.storage_location}
											<div class="badge badge-outline">
												{inv.storage_location.name} ({inv.quantity}x {inv.treatment || 'nonfoil'})
											</div>
										{/if}
									{/each}
								</div>
							</div>
						{/if}
					{:else}
						<p class="opacity-70">You don't own this printing yet.</p>
					{/if}

					<!-- Other printings inventory -->
					{#if card.inventory.other_printings.length > 0}
						<div class="divider"></div>
						<h4 class="font-semibold">Other Printings You Own</h4>
						<p class="text-sm opacity-70">
							{card.inventory.other_printings.reduce((sum, i) => sum + i.quantity, 0)} copies across other
							printings
						</p>
					{/if}
				</div>
			</div>

			<!-- Other Printings -->
			{#if otherPrintings.length > 0}
				<div class="card bg-base-200 shadow-lg">
					<div class="card-body">
						<h3 class="card-title text-lg">Other Printings ({otherPrintings.length})</h3>
						<div class="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-4">
							{#each otherPrintings as printing (printing.id)}
								<a
									href="/cards/{printing.id}"
									class="card bg-base-100 hover:bg-base-300 transition-colors cursor-pointer">
									{#if printing.image_uri}
										<figure class="px-2 pt-2">
											<img
												src={printing.image_uri}
												alt={printing.name}
												class="rounded-lg"
												loading="lazy" />
										</figure>
									{/if}
									<div class="card-body p-3">
										<p class="text-sm font-medium">{printing.set_name}</p>
										{#if printing.inventory.total_quantity > 0}
											<div class="badge badge-primary badge-sm">
												{printing.inventory.total_quantity} owned
											</div>
										{/if}
									</div>
								</a>
							{/each}
						</div>
					</div>
				</div>
			{/if}
		</div>
	</div>
</div>
