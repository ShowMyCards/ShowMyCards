<script lang="ts">
	import {
		Modal,
		FormField,
		inventoryApi,
		notifications,
		type Inventory,
		type StorageLocationWithCount
	} from '$lib';

	interface Props {
		open: boolean;
		inventory: Inventory | null;
		availableQuantity: number;
		currentLocationId?: number;
		locations: StorageLocationWithCount[];
		onClose: () => void;
		onComplete?: () => void;
	}

	let {
		open,
		inventory,
		availableQuantity,
		currentLocationId,
		locations,
		onClose,
		onComplete
	}: Props = $props();

	let quantity = $state(1);
	let selectedLocation = $state<number | 'unassigned' | undefined>(undefined);
	let isMoving = $state(false);

	// Reset the form whenever a different stack is opened.
	$effect(() => {
		if (inventory) {
			quantity = availableQuantity;
			selectedLocation = undefined;
		}
	});

	// Destinations exclude the stack's current location.
	const destinations = $derived(locations.filter((loc) => loc.id !== currentLocationId));

	async function handleMove() {
		if (!inventory || selectedLocation === undefined) return;

		isMoving = true;
		try {
			const destinationId = selectedLocation === 'unassigned' ? undefined : selectedLocation;
			const moved = quantity;
			await inventoryApi.splitMove(inventory.id, moved, destinationId);
			notifications.success(`Moved ${moved} ${moved === 1 ? 'copy' : 'copies'}`);
			onComplete?.();
			onClose();
		} catch (error) {
			notifications.error(error instanceof Error ? error.message : 'Failed to move copies');
		} finally {
			isMoving = false;
		}
	}
</script>

<Modal {open} {onClose} title="Move copies" boxClass="max-w-md">
	{#if inventory}
		<div class="space-y-4">
			<FormField
				label="Quantity to move"
				id="split-move-quantity"
				type="number"
				min={1}
				max={availableQuantity}
				bind:value={quantity}
				helper={`${availableQuantity} ${availableQuantity === 1 ? 'copy' : 'copies'} available`}
				required />

			<FormField label="Destination" id="split-move-destination" required>
				<select
					id="split-move-destination"
					bind:value={selectedLocation}
					disabled={isMoving}
					class="select select-bordered w-full">
					<option value={undefined}>Select location...</option>
					{#if currentLocationId !== undefined}
						<option value="unassigned">📥 Unassigned</option>
					{/if}
					{#each destinations as location (location.id)}
						<option value={location.id}>
							{location.storage_type === 'Binder' ? '📖' : '📦'}
							{location.name}
						</option>
					{/each}
				</select>
			</FormField>
		</div>

		<div class="modal-action">
			<button type="button" class="btn btn-ghost" onclick={onClose} disabled={isMoving}>
				Cancel
			</button>
			<button
				type="button"
				class="btn btn-primary"
				onclick={handleMove}
				disabled={isMoving ||
					selectedLocation === undefined ||
					quantity < 1 ||
					quantity > availableQuantity}>
				{#if isMoving}
					<span class="loading loading-spinner loading-xs"></span>
				{/if}
				Move
			</button>
		</div>
	{/if}
</Modal>
