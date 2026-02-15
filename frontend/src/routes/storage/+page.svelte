<script lang="ts">
	import { browser } from '$app/environment';
	import { enhance } from '$app/forms';
	import {
		PageHeader,
		Modal,
		EmptyState,
		FormField,
		StorageLocationRow,
		notifications,
		getActionError,
	} from '$lib';
	import type { PageData } from './$types';
	import { Plus } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// Derive storage locations from server data
	const storageLocations = $derived(data.locations || []);

	let showCreateModal = $state(false);
	let isCreating = $state(false);

	// Create form state
	let createFormValues = $state({
		name: '',
		storage_type: 'Box' as 'Box' | 'Binder'
	});

	// Display load error if present (browser only)
	// Only show error once per page load
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});
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

	{#if storageLocations.length === 0}
		<EmptyState message="No storage locations yet">
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				Create your first location
			</button>
		</EmptyState>
	{:else}
		<div class="card bg-base-200 shadow-lg">
			<div class="card-body p-0">
				<div class="divide-y divide-base-300">
					{#each storageLocations as location (location.id)}
						<StorageLocationRow {location} />
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
