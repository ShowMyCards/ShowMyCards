<script lang="ts">
	import type { ListSummary } from '$lib';
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { Modal, FormField, notifications, getActionError } from '$lib';
	import { ListTodo, Pencil, Trash2 } from '@lucide/svelte';

	let { list }: { list: ListSummary } = $props();

	const progressColor = $derived.by(() => {
		if (list.completion_percentage >= 67) return 'progress-success';
		if (list.completion_percentage >= 34) return 'progress-warning';
		return 'progress-error';
	});

	// Edit state
	let showEditModal = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let isSaving = $state(false);

	// Delete confirmation state
	let showDeleteModal = $state(false);
	let isDeleting = $state(false);

	function startEdit() {
		editName = list.name;
		editDescription = list.description;
		showEditModal = true;
	}
</script>

<div class="card bg-base-200 shadow-lg">
	<div class="card-body">
		<div class="flex items-start justify-between gap-4">
			<!-- Icon -->
			<div class="shrink-0 mt-1">
				<ListTodo class="w-6 h-6 text-primary" />
			</div>

			<!-- Content -->
			<div class="flex-1 min-w-0">
				<h3 class="font-semibold text-lg truncate">{list.name}</h3>
				{#if list.description}
					<p class="text-sm opacity-70 mt-1">{list.description}</p>
				{/if}

				<!-- Stats -->
				<div class="mt-3 space-y-2">
					<div class="text-sm">
						<span class="opacity-70"
							>{list.total_items} item{list.total_items === 1 ? '' : 's'}</span>
						<span class="opacity-50 mx-2">•</span>
						<span class="opacity-70">{list.completion_percentage}% complete</span>
					</div>

					<!-- Progress Bar -->
					<progress
						class="progress {progressColor} w-full"
						value={list.total_cards_collected}
						max={list.total_cards_wanted}></progress>

					<div class="text-xs opacity-60">
						{list.total_cards_collected} / {list.total_cards_wanted} cards collected
					</div>
				</div>
			</div>

			<!-- Actions -->
			<div class="shrink-0 flex items-center gap-2">
				<button onclick={startEdit} class="btn bg-base-100 btn-sm" title="Edit this list">
					<Pencil class="w-4 h-4" />
				</button>
				<button
					onclick={() => (showDeleteModal = true)}
					class="btn bg-base-100 btn-sm text-error"
					title="Delete this list">
					<Trash2 class="w-4 h-4" />
				</button>
				<a href={resolve(`/lists/${list.id}`)} class="btn btn-primary btn-sm"> Browse </a>
			</div>
		</div>
	</div>
</div>

<!-- Edit Modal -->
<Modal open={showEditModal} onClose={() => (showEditModal = false)} title="Edit List">
	<form
		method="POST"
		action="/lists?/update"
		use:enhance={() => {
			isSaving = true;
			return async ({ result, update }) => {
				isSaving = false;

				// Refresh the lists view first
				await update();

				if (result.type === 'success') {
					showEditModal = false;
					notifications.success('List updated successfully!');
				} else if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to update list'));
				}
			};
		}}>
		<input type="hidden" name="id" value={list.id} />

		<div class="space-y-4">
			<FormField
				label="Name"
				id="edit-name-{list.id}"
				name="name"
				placeholder="e.g., Commander Staples"
				bind:value={editName}
				helper="A descriptive name for this list"
				required />

			<FormField
				label="Description"
				id="edit-description-{list.id}"
				name="description"
				helper="Optional description">
				<textarea
					id="edit-description-{list.id}"
					name="description"
					bind:value={editDescription}
					class="textarea textarea-bordered w-full"
					rows="3"
					placeholder="What are you collecting?"></textarea>
			</FormField>
		</div>

		<div class="modal-action">
			<button
				type="button"
				onclick={() => (showEditModal = false)}
				disabled={isSaving}
				class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" disabled={isSaving} class="btn btn-primary">
				{#if isSaving}
					<span class="loading loading-spinner loading-sm"></span>
					Saving...
				{:else}
					Save
				{/if}
			</button>
		</div>
	</form>
</Modal>

<!-- Delete Confirmation Modal -->
<Modal open={showDeleteModal} onClose={() => (showDeleteModal = false)} title="Delete List">
	<p>
		Are you sure you want to delete <span class="font-semibold">{list.name}</span>? This will also
		remove all {list.total_items} item{list.total_items === 1 ? '' : 's'} in the list. This action cannot
		be undone.
	</p>

	<form
		method="POST"
		action="/lists?/delete"
		use:enhance={() => {
			isDeleting = true;
			return async ({ result, update }) => {
				isDeleting = false;

				// Refresh the lists view first
				await update();

				if (result.type === 'success') {
					showDeleteModal = false;
					notifications.success('List deleted successfully!');
				} else if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to delete list'));
				}
			};
		}}>
		<input type="hidden" name="id" value={list.id} />

		<div class="modal-action">
			<button
				type="button"
				onclick={() => (showDeleteModal = false)}
				disabled={isDeleting}
				class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" disabled={isDeleting} class="btn btn-error">
				{#if isDeleting}
					<span class="loading loading-spinner loading-sm"></span>
					Deleting...
				{:else}
					Delete
				{/if}
			</button>
		</div>
	</form>
</Modal>
