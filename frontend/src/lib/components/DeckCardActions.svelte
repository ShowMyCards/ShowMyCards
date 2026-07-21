<script lang="ts">
	import type { EnrichedDeckItem } from '$lib';
	import { notifications, getActionError } from '$lib';
	import { invalidateAll } from '$app/navigation';
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

	// Writable derived: seeded from the prop, editable, resets on fresh data.
	let qty = $derived(item.desired_quantity);
	const qtyChanged = $derived(qty !== item.desired_quantity && qty >= 1);

	let saving = $state(false);
	let removing = $state(false);

	async function post(action: string, fields: Record<string, string>): Promise<boolean> {
		const formData = new FormData();
		for (const [key, value] of Object.entries(fields)) {
			formData.append(key, value);
		}
		const response = await fetch(`?/${action}`, { method: 'POST', body: formData });
		const result = deserialize(await response.text());
		if (result.type === 'success') {
			await invalidateAll();
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

	async function changeZone(event: Event) {
		const target = event.currentTarget as HTMLSelectElement;
		const newZone = target.value;
		if (newZone === item.zone) return;
		saving = true;
		const ok = await post('updateItem', { item_id: String(item.id), zone: newZone });
		if (!ok) target.value = item.zone;
		saving = false;
	}

	async function remove() {
		if (removing) return;
		removing = true;
		try {
			const ok = await post('deleteItem', { item_id: String(item.id) });
			if (ok) notifications.success(`Removed ${item.name || 'card'} from deck`);
		} finally {
			removing = false;
		}
	}
</script>

<details class="dropdown dropdown-end">
	<summary
		class="btn btn-ghost btn-xs btn-square"
		title="Card actions"
		aria-label="Card actions for {item.name || 'card'}">
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
				value={item.zone}
				onchange={changeZone}
				disabled={saving}
				class="select select-bordered select-xs"
				aria-label="Zone">
				{#each ZONES as zone (zone.value)}
					<option value={zone.value}>{zone.label}</option>
				{/each}
			</select>
		</div>

		<button
			onclick={remove}
			disabled={removing}
			class="btn btn-ghost btn-xs text-error w-full justify-start">
			<Trash2 class="h-3 w-3" />
			Remove from deck
		</button>
	</div>
</details>
