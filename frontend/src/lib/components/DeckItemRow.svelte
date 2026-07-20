<script lang="ts">
	import type { EnrichedDeckItem } from '$lib';
	import { TreatmentBadge, notifications } from '$lib';
	import { invalidateAll } from '$app/navigation';
	import { deserialize } from '$app/forms';
	import { getActionError } from '$lib';
	import { Trash2, Check } from '@lucide/svelte';

	let {
		item,
		deckWants,
		spansZones = false,
		showAvailability = true,
		editable = true
	}: {
		/** The enriched item. `owned`, `under_owned`, `over_committed` are Oracle-level. */
		item: EnrichedDeckItem;
		/** Total desired quantity for this Oracle across all demand zones (rollup). */
		deckWants: number;
		/** True when this Oracle appears in more than one demand zone of the deck. */
		spansZones?: boolean;
		/**
		 * Whether to render the availability readout on this row. The parent shows it
		 * once per Oracle (on the first demand-zone row) so a split-zone card reads its
		 * signal once and correctly, not "short on both rows".
		 */
		showAvailability?: boolean;
		editable?: boolean;
	} = $props();

	const ZONES = [
		{ value: 'command', label: 'Command' },
		{ value: 'main', label: 'Main' },
		{ value: 'side', label: 'Side' },
		{ value: 'maybe', label: 'Maybe' }
	];

	// Maybe-board items never count as demand, so they carry no availability signal.
	const isDemand = $derived(item.zone !== 'maybe');

	// Writable derived: seeded from the prop and editable via the input, it resets
	// automatically when the row re-renders with fresh data after an update.
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
		if (result.type === 'failure') {
			notifications.error(getActionError(result.data, 'Update failed'));
		} else {
			notifications.error('Update failed');
		}
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
		if (!ok) {
			// Restore the previous selection when the change was rejected (e.g. a collision).
			target.value = item.zone;
		}
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

<div class="flex flex-wrap items-center gap-3 py-2 border-b border-base-300 last:border-b-0">
	<!-- Identity -->
	<div class="flex-1 min-w-48">
		<div class="flex items-center gap-2 flex-wrap">
			<span class="font-medium">{item.name || 'Unknown card'}</span>
			{#if item.treatment}
				<TreatmentBadge treatment={item.treatment} finishes={item.finishes ?? []} size="xs" />
			{/if}
			{#if item.scryfall_id}
				<span class="badge badge-outline badge-xs" title="Pinned to a specific printing">
					pinned
				</span>
			{/if}
		</div>
		{#if item.set_name}
			<div class="text-xs opacity-60">
				{item.set_name}
				{#if item.collector_number}· #{item.collector_number}{/if}
			</div>
		{/if}
	</div>

	<!-- Availability (rendered once per Oracle by the parent) -->
	{#if isDemand && showAvailability}
		<div class="flex items-center gap-2 flex-wrap text-sm">
			<span class="opacity-70" data-testid="availability">
				own {item.owned} / deck wants {deckWants}{spansZones ? ' across zones' : ''}
			</span>
			{#if item.under_owned}
				<span class="badge badge-warning badge-sm" data-testid="under-owned-badge">
					Under-owned
				</span>
			{:else if item.over_committed}
				<span class="badge badge-info badge-sm" data-testid="over-committed-badge">
					Over-committed
				</span>
			{/if}
		</div>
	{:else if isDemand}
		<!-- Secondary row of a split-zone card: don't repeat the badge, just cross-reference. -->
		<div class="text-xs opacity-50" data-testid="rollup-note">
			counts toward {deckWants} across zones
		</div>
	{:else}
		<div class="text-sm opacity-70">own {item.owned}</div>
	{/if}

	<!-- Controls -->
	{#if editable}
		<div class="flex items-center gap-2">
			<label class="flex items-center gap-1 text-xs">
				<span class="sr-only">Desired quantity</span>
				<input
					type="number"
					min="1"
					bind:value={qty}
					class="input input-bordered input-sm w-16"
					aria-label="Desired quantity" />
			</label>
			{#if qtyChanged}
				<button
					onclick={saveQuantity}
					disabled={saving}
					class="btn btn-sm btn-primary btn-square"
					title="Save quantity"
					aria-label="Save quantity">
					<Check class="w-4 h-4" />
				</button>
			{/if}

			<select
				value={item.zone}
				onchange={changeZone}
				disabled={saving}
				class="select select-bordered select-sm"
				aria-label="Zone">
				{#each ZONES as zone (zone.value)}
					<option value={zone.value}>{zone.label}</option>
				{/each}
			</select>

			<button
				onclick={remove}
				disabled={removing}
				class="btn btn-sm btn-square bg-base-100 text-error"
				title="Remove from deck"
				aria-label="Remove from deck">
				<Trash2 class="w-4 h-4" />
			</button>
		</div>
	{:else}
		<div class="text-sm opacity-70">×{item.desired_quantity}</div>
	{/if}
</div>
