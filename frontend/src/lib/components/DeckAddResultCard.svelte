<script lang="ts">
	import type { EnhancedCardResult } from '#lib';
	import { notifications, getActionError } from '#lib';
	import { refreshAll } from '$app/navigation';
	import { deserialize } from '$app/forms';
	import { Info, Plus } from '@lucide/svelte';

	let { card }: { card: EnhancedCardResult } = $props();

	const ZONES = [
		{ value: 'command', label: 'Command' },
		{ value: 'main', label: 'Main' },
		{ value: 'side', label: 'Side' },
		{ value: 'maybe', label: 'Maybe' }
	];

	let zone = $state('main');
	let qty = $state(1);
	let pin = $state(false);

	const PIN_HELP =
		'Only this exact printing counts toward the deck. Leave it unpinned and any printing of the card you own fills the slot.';
	let adding = $state(false);

	const imageUri = $derived(card.image_uri ?? '');

	async function add() {
		if (adding) return;
		adding = true;

		// treatment is left blank (any finish): finish is not part of the M1
		// allocation math, so pinning a printing is enough identity for now.
		const item = {
			oracle_id: card.oracle_id,
			scryfall_id: pin ? card.id : '',
			treatment: '',
			desired_quantity: qty,
			zone
		};

		const formData = new FormData();
		formData.append('items', JSON.stringify([item]));

		try {
			const response = await fetch('?/addItems', { method: 'POST', body: formData });
			const result = deserialize(await response.text());
			if (result.type === 'success') {
				await refreshAll();
				// Tokens and other extras are stored in Other whatever zone was picked,
				// so name the zone the server reports.
				const stored = result.data?.zones;
				const used = Array.isArray(stored) && stored.length > 0 ? (stored as string[]) : [zone];
				const zoneLabel = used
					.map((z) => ZONES.find((o) => o.value === z)?.label ?? (z === 'other' ? 'Other' : z))
					.join(', ');
				notifications.success(`Added ${card.name} to ${zoneLabel}`);
			} else if (result.type === 'failure') {
				notifications.error(getActionError(result.data, 'Failed to add card'));
			} else {
				notifications.error('Failed to add card');
			}
		} catch (error) {
			notifications.error('Error adding card: ' + error);
		} finally {
			adding = false;
		}
	}
</script>

<div class="card bg-base-200 p-3 shadow flex flex-col gap-3">
	<div class="flex gap-3">
		{#if imageUri}
			<img src={imageUri} alt={card.name} loading="lazy" class="w-16 rounded shrink-0" />
		{:else}
			<div class="w-16 aspect-5/7 bg-base-300 rounded flex items-center justify-center shrink-0">
				<span class="text-[10px] opacity-50 text-center">No image</span>
			</div>
		{/if}
		<div class="min-w-0">
			<div class="font-medium truncate">{card.name}</div>
			<div class="text-xs opacity-60">{card.set_name} · #{card.collector_number}</div>
			<div class="text-xs opacity-50 uppercase">{card.set_code}</div>
		</div>
	</div>

	<div class="flex flex-wrap items-end gap-2">
		<label class="flex flex-col gap-1 text-xs">
			<span class="opacity-70">Zone</span>
			<select bind:value={zone} class="select select-bordered select-sm" aria-label="Zone">
				{#each ZONES as z (z.value)}
					<option value={z.value}>{z.label}</option>
				{/each}
			</select>
		</label>

		<label class="flex flex-col gap-1 text-xs">
			<span class="opacity-70">Qty</span>
			<input
				type="number"
				min="1"
				bind:value={qty}
				class="input input-bordered input-sm w-16"
				aria-label="Desired quantity" />
		</label>

		<label
			class="tooltip tooltip-top flex items-center gap-1 text-xs cursor-pointer"
			data-tip={PIN_HELP}>
			<input
				type="checkbox"
				bind:checked={pin}
				class="checkbox checkbox-sm"
				aria-describedby="pin-help-{card.id}" />
			<span class="opacity-70">Pin printing</span>
			<Info class="h-3 w-3 opacity-50" aria-hidden="true" />
			<span id="pin-help-{card.id}" class="sr-only">{PIN_HELP}</span>
		</label>

		<button onclick={add} disabled={adding} class="btn btn-sm btn-primary ml-auto">
			<Plus class="w-4 h-4" />
			{adding ? 'Adding...' : 'Add'}
		</button>
	</div>
</div>
