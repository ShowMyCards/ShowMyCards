<script lang="ts">
	import { getDisplayName, isFoilTreatment, SCRYFALL_LANGUAGES, type EnrichedDeckItem } from '#lib';
	import { Pin, Sparkles } from '@lucide/svelte';
	import CardImage from './CardImage.svelte';
	import ManaCost from './ManaCost.svelte';
	import DeckCardActions from './DeckCardActions.svelte';
	import { groupCards, type DeckGroupMode, type DeckSortMode } from '#lib/utils/deck-grouping.ts';
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

	// An empty treatment means "any finish", which isFoilTreatment would count as foil.
	function isFoil(item: EnrichedDeckItem): boolean {
		return !!item.treatment && isFoilTreatment(item.treatment);
	}

	function pinnedTitle(item: EnrichedDeckItem): string {
		const where = [
			item.set_code?.toUpperCase(),
			item.collector_number && `#${item.collector_number}`
		]
			.filter(Boolean)
			.join(' ');
		const lang =
			item.language &&
			(SCRYFALL_LANGUAGES.find((l) => l.code === item.language)?.name ??
				item.language.toUpperCase());
		const printing = [where || 'a specific printing', lang].filter(Boolean).join(' · ');
		return `Pinned to ${printing}; other printings don't count for this slot`;
	}

	function cardHref(item: EnrichedDeckItem): string | undefined {
		return item.printing_id ? resolve(`cards/${item.printing_id}`) : undefined;
	}
</script>

{#snippet cardImage(item: EnrichedDeckItem)}
	{@const displayName = getDisplayName(item)}
	{#if item.image_uri}
		<CardImage
			src={item.image_uri}
			alt={displayName}
			isFoil={isFoil(item)}
			class="aspect-5/7 shadow" />
	{:else}
		<div
			class="bg-base-300 flex aspect-5/7 items-center justify-center rounded-xl p-2 text-center text-xs opacity-60">
			{displayName || 'Unknown card'}
		</div>
	{/if}
{/snippet}

{#snippet visualCard(item: EnrichedDeckItem)}
	{@const href = cardHref(item)}
	{#if href}
		<a
			href={resolve(`cards/${item.printing_id}`)}
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
						<div
							class="hover:bg-base-300/70 even:bg-base-200/60 first:rounded-t-box last:rounded-b-box flex items-center gap-2 px-3 py-1 text-sm transition-colors">
							<span class="w-5 shrink-0 text-right tabular-nums {qtyClass(item)}">
								{item.desired_quantity}
							</span>
							{#if href}
								<a
									href={resolve(`cards/${item.printing_id}`)}
									class="hover:text-primary flex-1 truncate"
									>{getDisplayName(item) || 'Unknown card'}</a>
							{:else}
								<span class="flex-1 truncate">{getDisplayName(item) || 'Unknown card'}</span>
							{/if}
							{#if isFoil(item)}
								{@const finish = item.treatment === 'etched' ? 'Etched' : 'Foil'}
								<span class="text-warning shrink-0" title={finish}>
									<Sparkles class="h-3 w-3" aria-label={finish} />
								</span>
							{/if}
							{#if item.scryfall_id}
								<span class="shrink-0 opacity-50" title={pinnedTitle(item)}>
									<Pin class="h-3 w-3" aria-label={pinnedTitle(item)} />
								</span>
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
