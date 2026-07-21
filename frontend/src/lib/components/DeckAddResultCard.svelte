<script lang="ts">
	import type { ScryfallCard } from '$lib';
	import { notifications, getActionError } from '$lib';
	import { invalidateAll } from '$app/navigation';
	import { deserialize } from '$app/forms';
	import { Plus } from '@lucide/svelte';

	let { card }: { card: ScryfallCard } = $props();

	const ZONES = [
		{ value: 'command', label: 'Command' },
		{ value: 'main', label: 'Main' },
		{ value: 'side', label: 'Side' },
		{ value: 'maybe', label: 'Maybe' }
	];

	let zone = $state('main');
	let qty = $state(1);
	let pin = $state(false);
	let adding = $state(false);

	const imageUri = $derived(card.image_uris?.small ?? card.image_uris?.normal ?? '');

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
				await invalidateAll();
				const zoneLabel = ZONES.find((z) => z.value === zone)?.label ?? zone;
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
			<div class="text-xs opacity-50 uppercase">{card.set}</div>
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
			class="flex items-center gap-1 text-xs cursor-pointer"
			title="Require this exact printing">
			<input type="checkbox" bind:checked={pin} class="checkbox checkbox-sm" />
			<span class="opacity-70">Pin printing</span>
		</label>

		<button onclick={add} disabled={adding} class="btn btn-sm btn-primary ml-auto">
			<Plus class="w-4 h-4" />
			{adding ? 'Adding...' : 'Add'}
		</button>
	</div>
</div>
