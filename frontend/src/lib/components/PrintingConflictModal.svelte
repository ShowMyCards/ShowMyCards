<script lang="ts">
	import Modal from './Modal.svelte';
	import type { StorageLocation } from '$lib';
	import type { ExistingPrintingInfo } from '$lib/types/api';

	interface Props {
		open: boolean;
		cardName: string;
		treatment: string;
		existingPrintings: ExistingPrintingInfo[];
		existingLocations: StorageLocation[];
		onClose: () => void;
		onChoose: (locationId: number | 'auto') => void;
	}

	let {
		open,
		cardName,
		treatment,
		existingPrintings,
		existingLocations,
		onClose,
		onChoose
	}: Props = $props();

	// Group existing printings by location for display
	const printingsByLocation = $derived.by(() => {
		const map = new Map<number | null, ExistingPrintingInfo[]>();
		for (const p of existingPrintings) {
			const locId = p.storage_location?.id ?? null;
			if (!map.has(locId)) {
				map.set(locId, []);
			}
			map.get(locId)!.push(p);
		}
		return map;
	});
</script>

<Modal {open} {onClose} title="You already own this card">
	{#snippet children()}
		<div class="space-y-4">
			<p class="text-sm opacity-80">
				You're adding <strong>{cardName}</strong> ({treatment}) to your inventory, but you already
				have other printings of this card:
			</p>

			<div class="bg-base-200 rounded-lg p-3 space-y-2">
				{#each existingLocations as location}
					{@const printings = printingsByLocation.get(location.id) || []}
					{@const totalQty = printings.reduce((sum, p) => sum + p.quantity, 0)}
					<div class="flex items-center justify-between text-sm">
						<span class="font-medium">{location.name}</span>
						<span class="badge badge-ghost">{totalQty} {totalQty === 1 ? 'copy' : 'copies'}</span>
					</div>
				{/each}
				{#if printingsByLocation.has(null)}
					{@const unassigned = printingsByLocation.get(null) || []}
					{@const totalQty = unassigned.reduce((sum, p) => sum + p.quantity, 0)}
					<div class="flex items-center justify-between text-sm opacity-70">
						<span class="italic">Unassigned</span>
						<span class="badge badge-ghost">{totalQty} {totalQty === 1 ? 'copy' : 'copies'}</span>
					</div>
				{/if}
			</div>

			<p class="text-sm">Where would you like to put this new printing?</p>
		</div>
	{/snippet}

	{#snippet actions()}
		<div class="flex flex-wrap gap-2 justify-end">
			<button type="button" class="btn btn-ghost" onclick={onClose}>Cancel</button>
			<button type="button" class="btn btn-outline" onclick={() => onChoose('auto')}>
				Auto-sort (use rules)
			</button>
			{#each existingLocations as location}
				<button type="button" class="btn btn-primary" onclick={() => onChoose(location.id)}>
					Send to {location.name}
				</button>
			{/each}
		</div>
	{/snippet}
</Modal>
