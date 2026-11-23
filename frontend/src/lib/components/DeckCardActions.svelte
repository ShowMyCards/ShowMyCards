<script lang="ts">
	import type { EnrichedDeckItem } from '#lib';
	import { getDisplayName, notifications, getActionError } from '#lib';
	import { refreshAll } from '$app/navigation';
	import { deserialize } from '$app/forms';
	import { ChevronDown, Trash2 } from '@lucide/svelte';

	let {
		item,
		deckWants,
		spansZones = false
	}: {
		item: EnrichedDeckItem;
		/** Total desired quantity for this Oracle across demand zones (rollup). */
		deckWants: number;
		spansZones?: boolean;
	} = $props();

	const ZONES = [
		{ value: 'command', label: 'Command' },
		{ value: 'main', label: 'Main' },
		{ value: 'side', label: 'Side' },
		{ value: 'maybe', label: 'Maybe' }
	];

	// Maybe-board items never count as demand, so they carry no availability signal.
	const isDemand = $derived(item.zone !== 'maybe');
	const displayName = $derived(getDisplayName(item) || 'card');

	// Writable derived: seeded from the prop, editable, resets on fresh data.
	let qty = $derived(item.desired_quantity);
	const qtyChanged = $derived(qty !== item.desired_quantity && qty >= 1);

	let saving = $state(false);
	let removing = $state(false);

	// Moving to another zone. With more than one copy we ask how many to move
	// (defaulting to all of them); a single copy moves as soon as a zone is picked.
	let moveZone = $derived<string>(item.zone);
	let moveQty = $derived(item.desired_quantity);
	const movePending = $derived(moveZone !== item.zone);
	const moveQtyValid = $derived(
		Number.isInteger(moveQty) && moveQty >= 1 && moveQty <= item.desired_quantity
	);
	const zoneLabel = (value: string) => ZONES.find((z) => z.value === value)?.label ?? value;

	// The <details> menu stays open until something closes it: all row menus
	// share a name so only one is open at a time, and outside clicks or Escape
	// close it here.
	let open = $state(false);
	let menu: HTMLDetailsElement | undefined = $state();

	function close() {
		open = false;
	}

	function onDocumentClick(event: MouseEvent) {
		if (open && menu && !menu.contains(event.target as Node)) close();
	}

	function onDocumentKeydown(event: KeyboardEvent) {
		if (open && event.key === 'Escape') {
			close();
			menu?.querySelector('summary')?.focus();
		}
	}

	async function post(action: string, fields: Record<string, string>): Promise<boolean> {
		const formData = new FormData();
		for (const [key, value] of Object.entries(fields)) {
			formData.append(key, value);
		}
		const response = await fetch(`?/${action}`, { method: 'POST', body: formData });
		const result = deserialize(await response.text());
		if (result.type === 'success') {
			await refreshAll();
			return true;
		}
		notifications.error(
			result.type === 'failure' ? getActionError(result.data, 'Update failed') : 'Update failed'
		);
		return false;
	}

	async function saveQuantity() {
		if (saving || !qtyChanged) return;
		saving = true;
		try {
			await post('updateItem', { item_id: String(item.id), desired_quantity: String(qty) });
		} finally {
			saving = false;
		}
	}

	function onZoneChange() {
		if (movePending && item.desired_quantity === 1) move();
	}

	function cancelMove() {
		moveZone = item.zone;
		moveQty = item.desired_quantity;
	}

	async function move() {
		if (saving || !movePending || !moveQtyValid) return;
		saving = true;
		try {
			const ok = await post('moveItem', {
				item_id: String(item.id),
				zone: moveZone,
				quantity: String(moveQty)
			});
			if (ok) {
				notifications.success(`Moved ${moveQty} ${displayName} to ${zoneLabel(moveZone)}`);
				close();
			} else {
				cancelMove();
			}
		} finally {
			saving = false;
		}
	}

	async function remove() {
		if (removing) return;
		removing = true;
		try {
			const ok = await post('deleteItem', { item_id: String(item.id) });
			if (ok) {
				notifications.success(`Removed ${displayName} from deck`);
				close();
			}
		} finally {
			removing = false;
		}
	}
</script>

<svelte:document onclick={onDocumentClick} onkeydown={onDocumentKeydown} />

<details class="dropdown dropdown-end" name="deck-card-actions" bind:open bind:this={menu}>
	<summary
		class="btn btn-ghost btn-xs btn-square"
		title="Card actions"
		aria-label="Card actions for {displayName}">
		<ChevronDown class="h-4 w-4" />
	</summary>
	<div
		class="dropdown-content bg-base-100 rounded-box border-base-300 z-20 w-60 border p-3 shadow-lg">
		<!-- Availability -->
		{#if isDemand}
			<div class="mb-1 text-xs">
				own {item.owned} / deck wants {deckWants}{spansZones ? ' across zones' : ''}
			</div>
			{#if item.under_owned}
				<span class="badge badge-warning badge-sm">Under-owned</span>
			{:else if item.over_committed}
				<span class="badge badge-info badge-sm">Over-committed</span>
			{:else}
				<span class="badge badge-ghost badge-sm">Owned</span>
			{/if}
		{:else}
			<div class="text-xs opacity-70">own {item.owned} · maybe board</div>
		{/if}

		<div class="divider my-2"></div>

		<!-- Quantity -->
		<div class="mb-2 flex items-center gap-2">
			<span class="w-12 text-xs opacity-70">Qty</span>
			<input
				type="number"
				min="1"
				bind:value={qty}
				class="input input-bordered input-xs w-16"
				aria-label="Desired quantity" />
			{#if qtyChanged}
				<button onclick={saveQuantity} disabled={saving} class="btn btn-primary btn-xs"
					>Save</button>
			{/if}
		</div>

		<!-- Zone -->
		<div class="mb-2 flex items-center gap-2">
			<span class="w-12 text-xs opacity-70">Zone</span>
			<select
				bind:value={moveZone}
				onchange={onZoneChange}
				disabled={saving}
				class="select select-bordered select-xs"
				aria-label="Zone">
				{#each ZONES as zone (zone.value)}
					<option value={zone.value}>{zone.label}</option>
				{/each}
			</select>
		</div>

		{#if movePending && item.desired_quantity > 1}
			<div class="bg-base-200 rounded-box mb-2 p-2">
				<div class="mb-2 flex items-center gap-2 text-xs">
					<span>Move</span>
					<input
						type="number"
						min="1"
						max={item.desired_quantity}
						bind:value={moveQty}
						class="input input-bordered input-xs w-14"
						aria-label="Quantity to move" />
					<span>of {item.desired_quantity} to {zoneLabel(moveZone)}</span>
				</div>
				<div class="flex gap-2">
					<button onclick={move} disabled={saving || !moveQtyValid} class="btn btn-primary btn-xs"
						>Move</button>
					<button onclick={cancelMove} disabled={saving} class="btn btn-ghost btn-xs"
						>Cancel</button>
				</div>
			</div>
		{/if}

		<button
			onclick={remove}
			disabled={removing}
			class="btn btn-ghost btn-xs text-error w-full justify-start">
			<Trash2 class="h-3 w-3" />
			Remove from deck
		</button>
	</div>
</details>
