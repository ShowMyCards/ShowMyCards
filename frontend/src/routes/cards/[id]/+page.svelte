<script lang="ts">
	import type { PageData } from './$types';
	import { PageHeader, getCardTreatmentName, getDisplayName } from '$lib';
	import { resolve } from '$app/paths';
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { ArrowLeft, ExternalLink } from '@lucide/svelte';
	import PriceLozenge from '$lib/components/PriceLozenge.svelte';
	import { SvelteMap } from 'svelte/reactivity';

	let { data }: { data: PageData } = $props();

	const card = $derived(data.card);
	const otherPrintings = $derived(data.otherPrintings);

	/**
	 * Back navigation target.
	 *
	 * Card detail pages can be reached from several places (search, a storage's
	 * inventory view, the recent/unassigned views, or a list), so a hardcoded
	 * "Back to Search" link sends users to the wrong place. We capture where the
	 * user arrived from and link back there instead.
	 *
	 * The origin is captured on the client whenever the user navigates here from
	 * a non-card page, and persisted in sessionStorage so it survives both a
	 * page reload and browsing between printings (each of which is a fresh
	 * navigation from another /cards/ page). When nothing has been captured we
	 * fall back to the search page, preserving the previous behaviour.
	 */
	const FALLBACK_BACK = { href: resolve('/search'), label: 'Back to Search' };
	const BACK_STORAGE_KEY = 'smc-card-back-target';

	let backPath = $state<string | null>(null);

	afterNavigate((navigation) => {
		const from = navigation.from?.url;
		if (from && !from.pathname.startsWith('/cards/')) {
			// Arrived from a non-card page: this is the origin to return to.
			backPath = from.pathname + from.search;
			try {
				sessionStorage.setItem(BACK_STORAGE_KEY, backPath);
			} catch {
				// sessionStorage may be unavailable (private mode); fall back to
				// in-memory state, which still covers same-session navigation.
			}
		} else if (backPath === null) {
			// Cold load or arrived from another printing: reuse the stored origin.
			try {
				backPath = sessionStorage.getItem(BACK_STORAGE_KEY);
			} catch {
				backPath = null;
			}
		}
	});

	/**
	 * Build a human-friendly label for a back-target path.
	 */
	function labelForPath(pathname: string): string {
		if (pathname.startsWith('/inventory')) return 'Back to Inventory';
		if (pathname.startsWith('/storage')) return 'Back to Storage';
		if (pathname.startsWith('/lists')) return 'Back to List';
		if (pathname.startsWith('/search')) return 'Back to Search';
		return 'Back';
	}

	// Only accept same-origin relative paths to avoid being redirected off-site.
	const safeBackPath = $derived(
		backPath && backPath.startsWith('/') && !backPath.startsWith('//') ? backPath : null
	);

	const back = $derived(
		safeBackPath
			? { href: safeBackPath, label: labelForPath(new URL(safeBackPath, page.url.origin).pathname) }
			: FALLBACK_BACK
	);

	// Group inventory by treatment
	const inventoryByTreatment = $derived.by(() => {
		// eslint-disable-next-line svelte/prefer-svelte-reactivity -- local variable in derived computation
		const map = new Map<string, number>();
		for (const inv of card.inventory.this_printing) {
			const treatment = inv.treatment || 'nonfoil';
			map.set(treatment, (map.get(treatment) || 0) + inv.quantity);
		}
		return map;
	});

	// Get available treatments
	const availableTreatments = $derived(card.finishes.length > 0 ? card.finishes : ['nonfoil']);

	// Group storage locations by (location + treatment), summing quantities.
	// Adding the same card to the same location creates separate inventory rows,
	// so consolidate them into a single badge per location/treatment.
	const storageByLocation = $derived.by(() => {
		const map = new SvelteMap<string, { name: string; treatment: string; quantity: number }>();
		for (const inv of card.inventory.this_printing) {
			if (!inv.storage_location) continue;
			const treatment = inv.treatment || 'nonfoil';
			const key = `${inv.storage_location_id}-${treatment}`;
			const existing = map.get(key);
			if (existing) {
				existing.quantity += inv.quantity;
			} else {
				map.set(key, { name: inv.storage_location.name, treatment, quantity: inv.quantity });
			}
		}
		return [...map.entries()].map(([key, value]) => ({ key, ...value }));
	});
</script>

<svelte:head>
	<title>{getDisplayName(card)} - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-6xl">
	<!-- Back button -->
	<!-- href is a runtime-captured in-app path; resolve() only accepts statically known route ids -->
	<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
	<a href={back.href} class="btn btn-ghost btn-sm mb-4">
		<ArrowLeft class="w-4 h-4" />
		{back.label}
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
			<PageHeader title={getDisplayName(card)} description={card.set_name || ''} />

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
									class="stat rounded-lg p-4"
									class:bg-base-100={!isFoil}
									class:bg-base-300={isFoil}>
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
						{#if storageByLocation.length > 0}
							<div class="mt-4">
								<h4 class="font-semibold mb-2">Storage Locations</h4>
								<div class="flex flex-wrap gap-2">
									{#each storageByLocation as location (location.key)}
										<div class="badge badge-outline">
											{location.name} ({location.quantity}x {location.treatment})
										</div>
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
								{@const ownedCount = printing.inventory.this_printing.reduce(
									(sum: number, i: { quantity: number }) => sum + i.quantity,
									0
								)}
								<a
									href={resolve(`/cards/${printing.id}`)}
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
										{#if ownedCount > 0}
											<div class="badge badge-primary badge-sm">
												{ownedCount} owned
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
