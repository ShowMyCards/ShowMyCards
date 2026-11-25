<script lang="ts">
	import { resolve } from '$app/paths';
	import type { Inventory } from '#lib';
	import { compareStorageLocations } from '#lib/utils/sort-storage.ts';

	let { inventory }: { inventory: Inventory[] } = $props();

	const locations = $derived.by(() => {
		const groups: { id: number | null; name: string; quantity: number; sortOrder?: number }[] = [];
		for (const item of inventory) {
			const id = item.storage_location_id ?? null;
			const existing = groups.find((location) => location.id === id);
			if (existing) {
				existing.quantity += item.quantity;
			} else {
				groups.push({
					id,
					name: id === null ? 'Unassigned' : (item.storage_location?.name ?? `Location ${id}`),
					quantity: item.quantity,
					sortOrder: item.storage_location?.sort_order
				});
			}
		}
		groups.sort(compareStorageLocations);
		return groups;
	});
</script>

<ul class="space-y-1 text-sm" aria-label="Storage locations">
	{#each locations as location (location.id)}
		<li>
			<a
				href={location.id === null
					? resolve('inventory/unassigned')
					: resolve(`inventory/${location.id}`)}
				class="link link-hover">{location.name}</a>
			<span class="opacity-70"
				>· {location.quantity} {location.quantity === 1 ? 'copy' : 'copies'}</span>
		</li>
	{/each}
</ul>
