<script lang="ts">
	import { browser } from '$app/environment';
	import type { PageData, ActionData } from './$types';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import {
		PageHeader,
		EmptyState,
		Modal,
		FormField,
		DeckItemRow,
		DeckAddResultCard,
		notifications,
		getActionError,
		type ScryfallCard,
		type EnrichedDeckItem
	} from '$lib';
	import { ArrowLeft, Pencil, Trash2, Search, Info } from '@lucide/svelte';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let searchQuery = $state('');
	let searching = $state(false);

	// Deck edit / delete state
	let showEditModal = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let isSaving = $state(false);
	let showDeleteModal = $state(false);
	let isDeleting = $state(false);

	function startEdit() {
		if (!data.deck) return;
		editName = data.deck.name;
		editDescription = data.deck.description ?? '';
		showEditModal = true;
	}

	// Surface load errors once (browser only).
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	$effect(() => {
		if (form && 'query' in form && form.query) {
			searchQuery = form.query as string;
		}
	});

	const searchResults = $derived<ScryfallCard[]>(
		form && 'searchResults' in form ? ((form.searchResults as ScryfallCard[]) ?? []) : []
	);

	const totalItems = $derived(
		data.items.command.length +
			data.items.main.length +
			data.items.side.length +
			data.items.maybe.length
	);

	// Roll demand-zone rows up per Oracle so a card split across zones (e.g. 3 main
	// + 1 side) shows its under-owned / over-committed signal ONCE, and correctly,
	// rather than appearing "short on both rows". under_owned / over_committed are
	// Oracle-level facts stamped on every row; we render them on the first
	// demand-zone row for each Oracle only.
	const rollup = $derived.by(() => {
		const demandOrdered = [...data.items.command, ...data.items.main, ...data.items.side];
		const wants: Record<string, number> = {};
		const zones: Record<string, Record<string, true>> = {};
		const seen: Record<string, true> = {};
		const showIds: number[] = [];
		for (const it of demandOrdered) {
			wants[it.oracle_id] = (wants[it.oracle_id] ?? 0) + it.desired_quantity;
			(zones[it.oracle_id] ??= {})[it.zone] = true;
			if (!seen[it.oracle_id]) {
				seen[it.oracle_id] = true;
				showIds.push(it.id);
			}
		}
		return { wants, zones, showIds };
	});

	function deckWantsFor(item: EnrichedDeckItem): number {
		return rollup.wants[item.oracle_id] ?? item.desired_quantity;
	}
	function spansZonesFor(item: EnrichedDeckItem): boolean {
		return Object.keys(rollup.zones[item.oracle_id] ?? {}).length > 1;
	}

	const ZONE_SECTIONS = [
		{ key: 'command' as const, title: 'Command' },
		{ key: 'main' as const, title: 'Main' },
		{ key: 'side' as const, title: 'Side' },
		{ key: 'maybe' as const, title: 'Maybe' }
	];
</script>

<svelte:head>
	<title>{data.deck?.name || 'Deck'} - ShowMyCards</title>
</svelte:head>

<div class="mx-auto px-4 py-8 max-w-5xl">
	{#if data.deck}
		<PageHeader title={data.deck.name} description={data.deck.description}>
			{#snippet actions()}
				<a href={resolve('/decks')} class="btn bg-base-100 btn-sm">
					<ArrowLeft class="w-4 h-4" />
					Back
				</a>
				<button onclick={startEdit} class="btn bg-base-100 btn-sm" title="Edit deck">
					<Pencil class="w-4 h-4" />
				</button>
				<button
					onclick={() => (showDeleteModal = true)}
					class="btn bg-base-100 btn-sm text-error"
					title="Delete deck">
					<Trash2 class="w-4 h-4" />
				</button>
			{/snippet}
		</PageHeader>

		<!-- Aggregate shortfall summary -->
		<div class="alert mb-4" class:alert-warning={data.items.aggregate_shortfall > 0}>
			{#if data.items.aggregate_shortfall > 0}
				<span>
					This deck needs <span class="font-semibold">{data.items.aggregate_shortfall}</span>
					more card{data.items.aggregate_shortfall === 1 ? '' : 's'} than you own.
				</span>
			{:else if totalItems > 0}
				<span>You own everything this deck needs.</span>
			{:else}
				<span>This deck is empty. Add cards below.</span>
			{/if}
		</div>

		<!-- M1 limitation copy -->
		<div class="text-xs opacity-60 flex items-start gap-2 mb-6">
			<Info class="w-4 h-4 shrink-0 mt-0.5" />
			<span>
				Availability is counted per card, ignoring finish — foil/etched versions are not yet
				factored into the owned-vs-needed math. Pinned printings are tracked but finish-aware
				availability is a later milestone.
			</span>
		</div>

		<!-- Zone sections -->
		{#if totalItems === 0}
			<EmptyState message="No cards in this deck yet">
				<p class="text-sm opacity-70">Use the search below to add cards.</p>
			</EmptyState>
		{:else}
			<div class="space-y-6 mb-8">
				{#each ZONE_SECTIONS as section (section.key)}
					{@const items = data.items[section.key]}
					{#if items.length > 0}
						<section>
							<h2 class="font-semibold text-lg mb-1">
								{section.title}
								<span class="opacity-50 text-sm font-normal">({items.length})</span>
							</h2>
							<div class="rounded-box border border-base-300 bg-base-100 px-4">
								{#each items as item (item.id)}
									<DeckItemRow
										{item}
										deckWants={deckWantsFor(item)}
										spansZones={spansZonesFor(item)}
										showAvailability={rollup.showIds.includes(item.id)} />
								{/each}
							</div>
						</section>
					{/if}
				{/each}
			</div>
		{/if}

		<!-- Add cards -->
		<section class="rounded-box border border-base-300 bg-base-100 p-6">
			<h2 class="card-title mb-3">Add Cards</h2>

			<form
				method="POST"
				action="?/search"
				use:enhance={() => {
					searching = true;
					return async ({ update }) => {
						await update({ reset: false });
						searching = false;
					};
				}}
				class="flex gap-2 mb-4">
				<input
					type="text"
					name="q"
					bind:value={searchQuery}
					placeholder="Search for cards to add..."
					class="input input-bordered flex-1"
					required />
				<button type="submit" disabled={searching} class="btn btn-primary">
					<Search class="w-4 h-4" />
					{searching ? 'Searching...' : 'Search'}
				</button>
			</form>

			{#if searchResults.length > 0}
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
					{#each searchResults as card (card.id)}
						<DeckAddResultCard {card} />
					{/each}
				</div>
			{:else if form && 'query' in form && form.query}
				<p class="text-sm opacity-70">No cards found for "{form.query}".</p>
			{/if}
		</section>
	{:else}
		<EmptyState message="Deck not found">
			<a href={resolve('/decks')} class="btn btn-primary">Back to Decks</a>
		</EmptyState>
	{/if}
</div>

<!-- Edit Deck Modal -->
<Modal open={showEditModal} onClose={() => (showEditModal = false)} title="Edit Deck">
	<form
		method="POST"
		action="?/updateDeck"
		use:enhance={() => {
			isSaving = true;
			return async ({ result, update }) => {
				isSaving = false;
				await update();
				if (result.type === 'success') {
					showEditModal = false;
					notifications.success('Deck updated successfully!');
				} else if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to update deck'));
				}
			};
		}}>
		<div class="space-y-4">
			<FormField
				label="Name"
				id="edit-deck-name"
				name="name"
				bind:value={editName}
				helper="A descriptive name for this deck"
				required />

			<FormField
				label="Description"
				id="edit-deck-description"
				name="description"
				helper="Optional description">
				<textarea
					id="edit-deck-description"
					name="description"
					bind:value={editDescription}
					class="textarea textarea-bordered w-full"
					rows="3"></textarea>
			</FormField>
		</div>

		<div class="modal-action">
			<button
				type="button"
				onclick={() => (showEditModal = false)}
				disabled={isSaving}
				class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" disabled={isSaving} class="btn btn-primary">
				{#if isSaving}
					<span class="loading loading-spinner loading-sm"></span>
					Saving...
				{:else}
					Save
				{/if}
			</button>
		</div>
	</form>
</Modal>

<!-- Delete Deck Modal -->
<Modal open={showDeleteModal} onClose={() => (showDeleteModal = false)} title="Delete Deck">
	<p>
		Are you sure you want to delete <span class="font-semibold">{data.deck?.name}</span>? This will
		also remove all {totalItems} item{totalItems === 1 ? '' : 's'} in the deck. This action cannot be
		undone.
	</p>

	<form
		method="POST"
		action="?/deleteDeck"
		use:enhance={() => {
			isDeleting = true;
			return async ({ result }) => {
				isDeleting = false;
				// A successful delete redirects (303) to /decks; enhance follows it.
				if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to delete deck'));
				}
			};
		}}>
		<div class="modal-action">
			<button
				type="button"
				onclick={() => (showDeleteModal = false)}
				disabled={isDeleting}
				class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" disabled={isDeleting} class="btn btn-error">
				{#if isDeleting}
					<span class="loading loading-spinner loading-sm"></span>
					Deleting...
				{:else}
					Delete
				{/if}
			</button>
		</div>
	</form>
</Modal>
