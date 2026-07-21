<script lang="ts">
	import type { EnrichedDeckItem } from '$lib';
	import ManaCost from './ManaCost.svelte';
	import DeckCardActions from './DeckCardActions.svelte';
	import { groupCards, type DeckGroupMode, type DeckSortMode } from '$lib/utils/deck-grouping';
	import { resolve } from '$app/paths';

	let {
		items,
		view,
		group,
		sort,
		getRollup
	}: {
		items: EnrichedDeckItem[];
		view: 'text' | 'grid' | 'stacks';
		group: DeckGroupMode;
		sort: DeckSortMode;
		/** Per-Oracle rollup for the actions dropdown (deck-wide demand). */
		getRollup: (item: EnrichedDeckItem) => { deckWants: number; spansZones: boolean };
	} = $props();

	const groups = $derived(groupCards(items, group, sort));

	function cardCount(list: EnrichedDeckItem[]): number {
		return list.reduce((sum, item) => sum + item.desired_quantity, 0);
	}

	// Colour the quantity by availability so shortfall reads at a glance without
	// cluttering the row (full status lives in the actions dropdown).
	function qtyClass(item: EnrichedDeckItem): string {
		if (item.zone === 'maybe') return 'opacity-60';
		if (item.under_owned) return 'text-warning';
		if (item.over_committed) return 'text-info';
		return 'opacity-60';
	}

	function cardHref(item: EnrichedDeckItem): string | undefined {
		return item.printing_id ? resolve(`/cards/${item.printing_id}`) : undefined;
	}
</script>

{#snippet cardImage(item: EnrichedDeckItem)}
	{#if item.image_uri}
		<img src={item.image_uri} alt={item.name} loading="lazy" class="w-full rounded-xl shadow" />
	{:else}
		<div
			class="bg-base-300 flex aspect-5/7 items-center justify-center rounded-xl p-2 text-center text-xs opacity-60">
			{item.name || 'Unknown card'}
		</div>
	{/if}
{/snippet}

{#snippet visualCard(item: EnrichedDeckItem)}
	{@const href = cardHref(item)}
	{#if href}
		<a
			href={resolve(`/cards/${item.printing_id}`)}
			class="block transition-transform hover:brightness-110">{@render cardImage(item)}</a>
	{:else}
		{@render cardImage(item)}
	{/if}
	<!-- Quantity badge: top-left, poking out, with shadow + ring so it stands off the
		 card's black border. -->
	<span
		class="badge badge-primary badge-sm ring-base-100/70 absolute -top-1.5 -left-1.5 px-2 font-bold shadow-md ring-2">
		{item.desired_quantity}
	</span>
	<div class="absolute top-1.5 right-1.5">
		<DeckCardActions {item} {...getRollup(item)} />
	</div>
{/snippet}

<div class={view === 'stacks' ? 'flex flex-wrap items-start gap-x-6 gap-y-4' : 'space-y-4'}>
	{#each groups as g (g.label)}
		<section class={view === 'stacks' ? 'w-44' : ''}>
			{#if g.label}
				<h4 class="mb-1 text-sm font-semibold">
					{g.label}
					<span class="font-normal opacity-50">({cardCount(g.items)})</span>
				</h4>
			{/if}

			{#if view === 'text'}
				<div class="rounded-box border-base-300 bg-base-100 divide-base-300 divide-y border">
					{#each g.items as item (item.id)}
						{@const href = cardHref(item)}
						<div class="flex items-center gap-2 px-3 py-1 text-sm">
							<span class="w-5 shrink-0 text-right tabular-nums {qtyClass(item)}">
								{item.desired_quantity}
							</span>
							{#if href}
								<a
									href={resolve(`/cards/${item.printing_id}`)}
									class="hover:text-primary flex-1 truncate">{item.name || 'Unknown card'}</a>
							{:else}
								<span class="flex-1 truncate">{item.name || 'Unknown card'}</span>
							{/if}
							<ManaCost cost={item.mana_cost} class="shrink-0" />
							<DeckCardActions {item} {...getRollup(item)} />
						</div>
					{/each}
				</div>
			{:else if view === 'grid'}
				<div class="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5">
					{#each g.items as item (item.id)}
						<div class="group relative">{@render visualCard(item)}</div>
					{/each}
				</div>
			{:else}
				<!-- Visual Stacks: cards overlap so only the title bar shows, except the last.
					 margin-top is a % of the column width; ~-122% reveals a ~13% title strip. -->
				<div class="flex flex-col">
					{#each g.items as item, i (item.id)}
						<div
							class="group relative transition-[margin] hover:z-10"
							style={i === 0 ? '' : 'margin-top: -122%'}>
							{@render visualCard(item)}
						</div>
					{/each}
				</div>
			{/if}
		</section>
	{/each}
</div>
