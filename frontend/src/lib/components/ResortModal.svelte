<script lang="ts">
	import { Modal, inventoryApi, notifications } from '$lib';
	import type { ResortResponse } from '$lib/types/api';
	import { RefreshCw, ArrowRight, CircleSlash } from '@lucide/svelte';

	interface Props {
		open: boolean;
		onclose: () => void;
		onComplete?: () => void;
	}

	let { open, onclose, onComplete }: Props = $props();

	let isResorting = $state(false);
	let result = $state<ResortResponse | null>(null);

	async function handleResort() {
		isResorting = true;
		result = null;

		try {
			// Resort all inventory items (no IDs = all)
			const response = await inventoryApi.resort();
			result = response;

			if (response.errors === 0) {
				notifications.success(`Re-sorted ${response.processed} items: ${response.updated} updated`);
			} else {
				notifications.warning(
					`Re-sorted ${response.processed} items: ${response.updated} updated, ${response.errors} errors`
				);
			}

			onComplete?.();
		} catch (error) {
			notifications.error(error instanceof Error ? error.message : 'Failed to re-sort inventory');
		} finally {
			isResorting = false;
		}
	}

	function handleClose() {
		if (!isResorting) {
			result = null;
			onclose();
		}
	}
</script>

<Modal {open} onClose={handleClose}>
	<h3 class="text-lg font-bold mb-4">Re-sort All Inventory</h3>

	{#if result === null}
		<p class="mb-4">
			This will re-evaluate all inventory items against your current sorting rules and update their
			storage locations accordingly.
		</p>

		<div class="bg-warning/10 border border-warning/30 rounded-lg p-4 mb-6">
			<p class="text-sm">
				<strong>Note:</strong> This operation may take some time depending on the size of your inventory.
				Items that no longer match any rule will have their storage location cleared.
			</p>
		</div>

		<div class="flex justify-end gap-2">
			<button class="btn btn-ghost" onclick={handleClose} disabled={isResorting}>Cancel</button>
			<button class="btn btn-primary" onclick={handleResort} disabled={isResorting}>
				{#if isResorting}
					<span class="loading loading-spinner loading-sm"></span>
					Re-sorting...
				{:else}
					<RefreshCw class="w-4 h-4" />
					Re-sort All
				{/if}
			</button>
		</div>
	{:else}
		<div class="space-y-4">
			<div class="stats stats-vertical lg:stats-horizontal w-full">
				<div class="stat">
					<div class="stat-title">Processed</div>
					<div class="stat-value text-primary">{result.processed}</div>
				</div>
				<div class="stat">
					<div class="stat-title">Updated</div>
					<div class="stat-value text-success">{result.updated}</div>
				</div>
				{#if result.errors > 0}
					<div class="stat">
						<div class="stat-title">Errors</div>
						<div class="stat-value text-error">{result.errors}</div>
					</div>
				{/if}
			</div>

			{#if result.movements && result.movements.length > 0}
				<div class="max-h-64 overflow-y-auto">
					<table class="table table-sm">
						<thead class="sticky top-0 bg-base-100">
							<tr>
								<th>Card</th>
								<th>From</th>
								<th></th>
								<th>To</th>
							</tr>
						</thead>
						<tbody>
							{#each result.movements as movement, i (i)}
								<tr>
									<td>
										<div class="font-medium">{movement.card_name}</div>
										{#if movement.treatment && movement.treatment !== 'nonfoil'}
											<div class="text-xs opacity-70">{movement.treatment}</div>
										{/if}
									</td>
									<td>
										{#if movement.from_location}
											<span class="badge badge-ghost badge-sm">{movement.from_location}</span>
										{:else}
											<span class="text-warning flex items-center gap-1">
												<CircleSlash class="w-3 h-3" />
												Unassigned
											</span>
										{/if}
									</td>
									<td class="text-center">
										<ArrowRight class="w-4 h-4 opacity-50" />
									</td>
									<td>
										{#if movement.to_location}
											<span class="badge badge-primary badge-sm">{movement.to_location}</span>
										{:else}
											<span class="text-warning flex items-center gap-1">
												<CircleSlash class="w-3 h-3" />
												Unassigned
											</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			{:else if result.updated === 0}
				<p class="text-sm opacity-70">No items needed to be moved based on current rules.</p>
			{/if}

			<div class="flex justify-end">
				<button class="btn btn-primary" onclick={handleClose}>Done</button>
			</div>
		</div>
	{/if}
</Modal>
