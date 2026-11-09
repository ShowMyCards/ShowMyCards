<script lang="ts">
	import { untrack } from 'svelte';
	import { decksApi, notifications, type DeckSummary, type DeckZone } from '$lib';
	import { buildDeckItems, type AddToDeckCard } from '$lib/utils/deck-add';
	import { LayersPlus, Plus } from '@lucide/svelte';
	import Modal from './Modal.svelte';

	interface Props {
		open: boolean;
		/** Cards to add. A single card gets a quantity input; bulk gets a copies toggle. */
		cards: AddToDeckCard[];
		/** Bulk mode (inventory selection): offer "all selected copies" vs "1 of each". */
		bulk?: boolean;
		onClose: () => void;
		onAdded?: (deckName: string) => void;
	}

	let { open, cards, bulk = false, onClose, onAdded }: Props = $props();

	const uid = $props.id();

	const ZONES: { value: DeckZone; label: string }[] = [
		{ value: 'command', label: 'Command' },
		{ value: 'main', label: 'Main' },
		{ value: 'side', label: 'Side' },
		{ value: 'maybe', label: 'Maybe' }
	];

	// Remember the last deck used so repeat adds are a single click. Per-viewer
	// convenience only, so storage failures (private mode etc.) are ignored.
	const LAST_DECK_KEY = 'smc-last-deck-id';

	function readLastDeck(): string {
		try {
			return localStorage.getItem(LAST_DECK_KEY) ?? '';
		} catch {
			return '';
		}
	}

	function rememberDeck(id: string) {
		try {
			localStorage.setItem(LAST_DECK_KEY, id);
		} catch {
			// ignore
		}
	}

	let decks = $state<DeckSummary[]>([]);
	let loadState = $state<'loading' | 'ready' | 'error'>('loading');
	let deckId = $state('');
	let zone = $state<DeckZone>('main');
	let qty = $state(1);
	let copiesMode = $state<'copies' | 'one'>('copies');
	let pin = $state(false);
	let saving = $state(false);

	let creatingDeck = $state(false);
	let newDeckName = $state('');
	let savingDeck = $state(false);

	const totalCopies = $derived(cards.reduce((sum, c) => sum + c.copies, 0));
	const single = $derived(!bulk && cards.length === 1 ? cards[0] : null);
	const previewNames = $derived(cards.slice(0, 3).map((c) => c.name));
	const selectedDeck = $derived(decks.find((d) => String(d.id) === deckId));
	const canSubmit = $derived(
		!!selectedDeck && cards.length > 0 && !saving && (bulk || (Number.isInteger(qty) && qty >= 1))
	);

	async function loadDecks() {
		loadState = 'loading';
		try {
			decks = await decksApi.list();
			const last = readLastDeck();
			if (decks.some((d) => String(d.id) === last)) {
				deckId = last;
			} else if (decks.length === 1) {
				deckId = String(decks[0].id);
			}
			creatingDeck = decks.length === 0;
			loadState = 'ready';
		} catch {
			loadState = 'error';
		}
	}

	$effect(() => {
		if (open) untrack(loadDecks);
	});

	async function createDeck() {
		const name = newDeckName.trim();
		if (!name || savingDeck) return;
		savingDeck = true;
		try {
			const created = await decksApi.create({ name, description: '' });
			decks = [
				{
					id: created.id,
					name: created.name,
					description: created.description ?? '',
					created_at: '',
					updated_at: '',
					total_items: 0,
					total_cards_demand: 0,
					aggregate_shortfall: 0
				},
				...decks
			];
			deckId = String(created.id);
			creatingDeck = false;
			newDeckName = '';
		} catch (e) {
			notifications.error(e instanceof Error ? e.message : 'Failed to create deck');
		} finally {
			savingDeck = false;
		}
	}

	function handleNewDeckKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter') {
			e.preventDefault();
			createDeck();
		} else if (e.key === 'Escape' && decks.length > 0) {
			e.preventDefault();
			creatingDeck = false;
		}
	}

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		if (!canSubmit || !selectedDeck) return;

		const deck = selectedDeck;
		const items = buildDeckItems(cards, {
			zone,
			pin,
			quantity: bulk ? (copiesMode === 'copies' ? 'copies' : 1) : qty
		});
		if (items.length === 0) return;

		saving = true;
		try {
			const response = await fetch(`/decks/${deck.id}/items`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ items })
			});
			if (!response.ok) {
				const data = await response.json().catch(() => ({}));
				throw new Error(data.error || 'Failed to add to deck');
			}

			rememberDeck(String(deck.id));
			const zoneLabel = ZONES.find((z) => z.value === zone)?.label ?? zone;
			notifications.success(
				single
					? `Added ${qty > 1 ? `${qty}× ` : ''}${single.name} to ${deck.name} (${zoneLabel})`
					: `Added ${items.length} card${items.length === 1 ? '' : 's'} to ${deck.name} (${zoneLabel})`
			);
			onAdded?.(deck.name);
			onClose();
		} catch (err) {
			notifications.error(err instanceof Error ? err.message : 'Failed to add to deck');
		} finally {
			saving = false;
		}
	}
</script>

<Modal {open} {onClose} title="Add to deck" boxClass="max-w-md">
	<form onsubmit={submit} class="space-y-5">
		<!-- What is being added -->
		{#if single}
			<div class="flex items-center gap-3 rounded-box bg-base-200 p-2">
				{#if single.image_uri}
					<img src={single.image_uri} alt="" class="w-12 shrink-0 rounded" />
				{/if}
				<div class="min-w-0">
					<div class="truncate font-semibold">{single.name}</div>
					{#if single.set_name}
						<div class="truncate text-sm opacity-70">{single.set_name}</div>
					{/if}
				</div>
			</div>
		{:else}
			<div class="rounded-box bg-base-200 p-3">
				<div class="font-semibold">
					{cards.length}
					{cards.length === 1 ? 'card' : 'cards'} · {totalCopies}
					{totalCopies === 1 ? 'copy' : 'copies'} selected
				</div>
				<div class="truncate text-sm opacity-70">
					{previewNames.join(', ')}{cards.length > 3 ? ` +${cards.length - 3} more` : ''}
				</div>
			</div>
		{/if}

		<!-- Deck -->
		<div class="flex flex-col gap-2">
			<label for="{uid}-deck" class="font-semibold">Deck</label>
			{#if loadState === 'loading'}
				<div class="flex items-center gap-2 text-sm opacity-70">
					<span class="loading loading-spinner loading-sm"></span>
					Loading decks…
				</div>
			{:else if loadState === 'error'}
				<div role="alert" class="alert alert-error alert-soft py-2 text-sm">
					<span>Couldn't load your decks.</span>
					<button type="button" class="btn btn-ghost btn-xs" onclick={loadDecks}>Retry</button>
				</div>
			{:else if creatingDeck}
				{#if decks.length === 0}
					<p class="text-sm opacity-70">You have no decks yet. Name your first one:</p>
				{/if}
				<div class="join w-full">
					<input
						id="{uid}-deck"
						class="input input-sm join-item flex-1"
						placeholder="New deck name"
						bind:value={newDeckName}
						onkeydown={handleNewDeckKeydown}
						disabled={savingDeck} />
					<button
						type="button"
						class="btn btn-sm btn-primary join-item"
						onclick={createDeck}
						disabled={!newDeckName.trim() || savingDeck}>
						{#if savingDeck}<span class="loading loading-spinner loading-xs"></span>{/if}
						Create
					</button>
					{#if decks.length > 0}
						<button
							type="button"
							class="btn btn-sm join-item"
							onclick={() => (creatingDeck = false)}>
							Cancel
						</button>
					{/if}
				</div>
			{:else}
				<div class="flex gap-2">
					<select id="{uid}-deck" class="select select-sm flex-1" bind:value={deckId}>
						<option value="" disabled>Select a deck…</option>
						{#each decks as deck (deck.id)}
							<option value={String(deck.id)}>{deck.name}</option>
						{/each}
					</select>
					<button
						type="button"
						class="btn btn-sm btn-ghost"
						onclick={() => (creatingDeck = true)}
						title="Create a new deck">
						<Plus class="h-4 w-4" />
						New
					</button>
				</div>
			{/if}
		</div>

		<!-- Zone -->
		<fieldset class="flex flex-col gap-2">
			<legend class="mb-2 font-semibold">Zone</legend>
			<div class="join">
				{#each ZONES as z (z.value)}
					<input
						type="radio"
						name="{uid}-zone"
						class="join-item btn btn-sm"
						aria-label={z.label}
						value={z.value}
						bind:group={zone} />
				{/each}
			</div>
		</fieldset>

		<!-- Quantity -->
		{#if bulk}
			<fieldset class="flex flex-col gap-2">
				<legend class="mb-2 font-semibold">Copies</legend>
				<div class="join">
					<input
						type="radio"
						name="{uid}-copies"
						class="join-item btn btn-sm"
						aria-label="All selected ({totalCopies})"
						value="copies"
						bind:group={copiesMode} />
					<input
						type="radio"
						name="{uid}-copies"
						class="join-item btn btn-sm"
						aria-label="1 of each"
						value="one"
						bind:group={copiesMode} />
				</div>
			</fieldset>
		{:else}
			<div class="flex flex-col gap-2">
				<label for="{uid}-qty" class="font-semibold">Quantity</label>
				<input id="{uid}-qty" type="number" min="1" class="input input-sm w-24" bind:value={qty} />
			</div>
		{/if}

		<!-- Pin printing -->
		<label class="flex cursor-pointer items-start gap-3">
			<input type="checkbox" class="checkbox checkbox-sm mt-0.5" bind:checked={pin} />
			<span class="text-sm">
				<span class="font-medium">Pin {single ? 'this printing' : 'these printings'}</span>
				<span class="block opacity-70">
					Otherwise any printing of the card fills the deck slot.
				</span>
			</span>
		</label>

		<div class="modal-action">
			<button type="button" class="btn btn-ghost" onclick={onClose}>Cancel</button>
			<button type="submit" class="btn btn-primary" disabled={!canSubmit}>
				{#if saving}
					<span class="loading loading-spinner loading-xs"></span>
				{:else}
					<LayersPlus class="h-4 w-4" />
				{/if}
				Add to deck
			</button>
		</div>
	</form>
</Modal>
