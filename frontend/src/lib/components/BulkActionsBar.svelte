<script lang="ts">
	import { selection, inventoryApi, notifications, type StorageLocation } from '$lib';
	import { X, FolderInput, Trash2, RefreshCw } from '@lucide/svelte';

	interface Props {
		locations: StorageLocation[];
		onComplete?: () => void;
	}

	let { locations, onComplete }: Props = $props();

	let isMoving = $state(false);
	let isDeleting = $state(false);
	let isResorting = $state(false);
	let showMoveDropdown = $state(false);
	let selectedLocation = $state<number | undefined>(undefined);

	const selectedCount = $derived(selection.count);

	async function handleMove() {
		if (selectedCount === 0) return;

		isMoving = true;
		try {
			const ids = selection.getSelected();
			const result = await inventoryApi.batchMove(ids, selectedLocation);
			notifications.success(`Moved ${result.updated} items`);
			selection.clear();
			showMoveDropdown = false;
			selectedLocation = undefined;
			onComplete?.();
		} catch (error) {
			notifications.error(error instanceof Error ? error.message : 'Failed to move items');
		} finally {
			isMoving = false;
		}
	}

	async function handleDelete() {
		if (selectedCount === 0) return;

		const confirmed = confirm(`Are you sure you want to delete ${selectedCount} items?`);
		if (!confirmed) return;

		isDeleting = true;
		try {
			const ids = selection.getSelected();
			const result = await inventoryApi.batchDelete(ids);
			notifications.success(`Deleted ${result.deleted} items`);
			selection.clear();
			onComplete?.();
		} catch (error) {
			notifications.error(error instanceof Error ? error.message : 'Failed to delete items');
		} finally {
			isDeleting = false;
		}
	}

	async function handleResort() {
		if (selectedCount === 0) return;

		isResorting = true;
		try {
			const ids = selection.getSelected();
			const result = await inventoryApi.resort(ids);
			notifications.success(
				`Re-sorted ${result.processed} items: ${result.updated} updated, ${result.errors} errors`
			);
			selection.clear();
			onComplete?.();
		} catch (error) {
			notifications.error(error instanceof Error ? error.message : 'Failed to re-sort items');
		} finally {
			isResorting = false;
		}
	}

	function handleClearSelection() {
		selection.clear();
		showMoveDropdown = false;
	}
</script>

{#if selectedCount > 0}
	<div
		class="fixed bottom-0 left-0 right-0 z-50 border-t border-base-300 bg-base-100 px-4 py-3 shadow-lg">
		<div class="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-4">
			<div class="flex items-center gap-4">
				<span class="font-medium">{selectedCount} selected</span>
				<button class="btn btn-ghost btn-sm" onclick={handleClearSelection}>
					<X class="h-4 w-4" />
					Clear
				</button>
			</div>

			<div class="flex flex-wrap items-center gap-2">
				{#if showMoveDropdown}
					<div class="flex items-center gap-2">
						<select
							class="select select-bordered select-sm"
							bind:value={selectedLocation}
							disabled={isMoving}
							aria-label="Select destination location">
							<option value={undefined}>Select location...</option>
							{#each locations as location}
								<option value={location.id}>
									{location.storage_type === 'Binder' ? '📖' : '📦'}
									{location.name}
								</option>
							{/each}
						</select>
						<button
							class="btn btn-primary btn-sm"
							onclick={handleMove}
							disabled={isMoving || selectedLocation === undefined}>
							{#if isMoving}
								<span class="loading loading-spinner loading-xs"></span>
							{/if}
							Move
						</button>
						<button
							class="btn btn-ghost btn-sm"
							onclick={() => {
								showMoveDropdown = false;
								selectedLocation = undefined;
							}}>
							Cancel
						</button>
					</div>
				{:else}
					<button
						class="btn btn-outline btn-sm"
						onclick={() => (showMoveDropdown = true)}
						disabled={isMoving || isDeleting || isResorting}>
						<FolderInput class="h-4 w-4" />
						Move to...
					</button>
				{/if}

				<button
					class="btn btn-outline btn-sm"
					onclick={handleResort}
					disabled={isMoving || isDeleting || isResorting}>
					{#if isResorting}
						<span class="loading loading-spinner loading-xs"></span>
					{:else}
						<RefreshCw class="h-4 w-4" />
					{/if}
					Re-sort
				</button>

				<button
					class="btn btn-error btn-outline btn-sm"
					onclick={handleDelete}
					disabled={isMoving || isDeleting || isResorting}>
					{#if isDeleting}
						<span class="loading loading-spinner loading-xs"></span>
					{:else}
						<Trash2 class="h-4 w-4" />
					{/if}
					Delete
				</button>
			</div>
		</div>
	</div>
{/if}
