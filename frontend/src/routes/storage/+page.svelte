<script lang="ts">
	import { deserialize, enhance } from '$app/forms';
	import { refreshAll } from '$app/navigation';
	import {
		PageHeader,
		Modal,
		EmptyState,
		FormField,
		Notification,
		StorageLocationRow,
		notifications,
		getActionError,
		type StorageLocation
	} from '#lib';
	import type { DragDropState } from '@thisux/sveltednd';
	import type { PageData } from './$types';
	import { Plus } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// Derive storage locations from server data, overlaid with any in-flight optimistic reorder
	const storageLocations = $derived(data.locations || []);
	let optimisticLocations = $state<StorageLocation[] | null>(null);
	const displayedLocations = $derived(optimisticLocations ?? storageLocations);

	let isSavingOrder = false;
	let pendingUpdates: Array<{ id: number; sort_order: number }> | null = null;

	function takePendingUpdates() {
		const next = pendingUpdates;
		pendingUpdates = null;
		return next;
	}

	let showCreateModal = $state(false);
	let isCreating = $state(false);

	// Create form state
	let createFormValues = $state({
		name: '',
		storage_type: 'Box' as 'Box' | 'Binder'
	});

	function applyReorder(fromIndex: number, toIndex: number) {
		if (
			fromIndex < 0 ||
			toIndex < 0 ||
			fromIndex >= displayedLocations.length ||
			toIndex >= displayedLocations.length ||
			fromIndex === toIndex
		) {
			return;
		}

		const next = [...displayedLocations];
		const [moved] = next.splice(fromIndex, 1);
		next.splice(toIndex, 0, moved);

		const updated = next.map((loc, idx) => ({
			...loc,
			sort_order: idx + 1
		}));

		optimisticLocations = updated;
		void persistOrder(updated.map((loc) => ({ id: loc.id, sort_order: loc.sort_order })));
	}

	function handleDrop(state: DragDropState<StorageLocation>) {
		const { draggedItem, targetContainer } = state;
		if (!draggedItem || targetContainer === undefined || targetContainer === null) return;

		const targetIndex = parseInt(targetContainer, 10);
		const draggedIndex = displayedLocations.findIndex((loc) => loc.id === draggedItem.id);
		if (Number.isNaN(targetIndex) || draggedIndex === -1 || draggedIndex === targetIndex) return;

		applyReorder(draggedIndex, targetIndex);
	}

	async function persistOrder(updates: Array<{ id: number; sort_order: number }>) {
		if (isSavingOrder) {
			pendingUpdates = updates;
			return;
		}

		isSavingOrder = true;
		let current: Array<{ id: number; sort_order: number }> | null = updates;

		try {
			while (current) {
				const batch = current;

				const formData = new FormData();
				formData.append('updates', JSON.stringify(batch));

				const response = await fetch('?/reorder', {
					method: 'POST',
					body: formData
				});

				const result = deserialize(await response.text());
				if (result.type !== 'success') {
					const errorMsg =
						result.type === 'failure'
							? getActionError(result.data, 'Failed to save storage location order')
							: 'Failed to save storage location order';
					notifications.error(errorMsg);
					await refreshAll().catch(() => {});
					break;
				}

				current = takePendingUpdates();
				if (current) {
					continue;
				}

				await refreshAll();
				current = takePendingUpdates();
			}
		} catch {
			notifications.error('Failed to save storage location order');
			await refreshAll().catch(() => {});
		} finally {
			optimisticLocations = null;
			pendingUpdates = null;
			isSavingOrder = false;
		}
	}
</script>

<svelte:head>
	<title>Storage Locations - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader title="Storage Locations" description="Manage where you store your card collection">
		{#snippet actions()}
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				<Plus class="w-4 h-4" />
				New Location
			</button>
		{/snippet}
	</PageHeader>

	{#if data.error}
		<Notification type="error">{data.error}</Notification>
	{:else if displayedLocations.length === 0}
		<EmptyState message="No storage locations yet">
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				Create your first location
			</button>
		</EmptyState>
	{:else}
		<div class="card bg-base-200 shadow-lg">
			<div class="card-body p-0">
				<div class="divide-y divide-base-300">
					{#each displayedLocations as location, index (location.id)}
						<StorageLocationRow
							{location}
							{index}
							total={displayedLocations.length}
							onDrop={handleDrop}
							onMoveUp={() => applyReorder(index, index - 1)}
							onMoveDown={() => applyReorder(index, index + 1)} />
					{/each}
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- Create Modal -->
<Modal
	open={showCreateModal}
	onClose={() => (showCreateModal = false)}
	title="Create Storage Location">
	<form
		method="POST"
		action="?/create"
		use:enhance={() => {
			isCreating = true;
			return async ({ result, update }) => {
				isCreating = false;

				// Update the page data first
				await update();

				// Then show notifications based on result
				if (result.type === 'success') {
					notifications.success('Storage location created successfully!');
					// Close modal on success
					showCreateModal = false;
					createFormValues = { name: '', storage_type: 'Box' };
				} else if (result.type === 'failure') {
					const errorMsg = getActionError(result.data, 'Failed to create storage location');
					notifications.error(errorMsg);
				}
			};
		}}>
		<div class="space-y-4">
			<FormField
				label="Name"
				id="create-name"
				name="name"
				placeholder="e.g., Living Room Shelf"
				bind:value={createFormValues.name}
				helper="A descriptive name for this storage location"
				required />

			<FormField label="Storage Type" id="create-type" required>
				<select
					id="create-type"
					name="storage_type"
					bind:value={createFormValues.storage_type}
					class="select select-bordered w-full">
					<option value="Box">Box</option>
					<option value="Binder">Binder</option>
				</select>
			</FormField>
		</div>

		<div class="modal-action">
			<button
				type="button"
				onclick={() => (showCreateModal = false)}
				disabled={isCreating}
				class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" disabled={isCreating} class="btn btn-primary">
				{#if isCreating}
					<span class="loading loading-spinner loading-sm"></span>
					Creating...
				{:else}
					Create
				{/if}
			</button>
		</div>
	</form>
</Modal>

<style>
	:global(.dragging) {
		opacity: 0.5;
		box-shadow:
			0 10px 15px -3px rgb(0 0 0 / 0.1),
			0 4px 6px -4px rgb(0 0 0 / 0.1);
	}

	:global(.drag-over) {
		background-color: hsl(var(--p) / 0.15);
		box-shadow: 0 0 0 2px hsl(var(--p) / 0.3);
		transform: scale(1.01);
	}
</style>
