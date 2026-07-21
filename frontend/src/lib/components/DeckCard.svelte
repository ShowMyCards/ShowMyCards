<script lang="ts">
	import type { DeckSummary } from '$lib';
	import { resolve } from '$app/paths';
	import { enhance } from '$app/forms';
	import { Modal, FormField, notifications, getActionError } from '$lib';
	import { Layers, Pencil, Trash2 } from '@lucide/svelte';

	let { deck }: { deck: DeckSummary } = $props();

	// Edit state
	let showEditModal = $state(false);
	let editName = $state('');
	let editDescription = $state('');
	let isSaving = $state(false);

	// Delete confirmation state
	let showDeleteModal = $state(false);
	let isDeleting = $state(false);

	function startEdit() {
		editName = deck.name;
		editDescription = deck.description;
		showEditModal = true;
	}
</script>

<div class="card bg-base-200 shadow-lg">
	<div class="card-body">
		<div class="flex items-start justify-between gap-4">
			<!-- Icon -->
			<div class="shrink-0 mt-1">
				<Layers class="w-6 h-6 text-primary" />
			</div>

			<!-- Content -->
			<div class="flex-1 min-w-0">
				<h3 class="font-semibold text-lg truncate">{deck.name}</h3>
				{#if deck.description}
					<p class="text-sm opacity-70 mt-1">{deck.description}</p>
				{/if}

				<!-- Stats -->
				<div class="mt-3 text-sm flex flex-wrap items-center gap-x-2 gap-y-1">
					<span class="opacity-70">
						{deck.total_items} item{deck.total_items === 1 ? '' : 's'}
					</span>
					<span class="opacity-50">•</span>
					<span class="opacity-70">
						{deck.total_cards_demand} card{deck.total_cards_demand === 1 ? '' : 's'}
					</span>
					{#if deck.aggregate_shortfall > 0}
						<span class="opacity-50">•</span>
						<span class="badge badge-warning badge-sm">
							Short {deck.aggregate_shortfall}
						</span>
					{:else if deck.total_cards_demand > 0}
						<span class="opacity-50">•</span>
						<span class="badge badge-success badge-sm">Fully owned</span>
					{/if}
				</div>
			</div>

			<!-- Actions -->
			<div class="shrink-0 flex items-center gap-2">
				<button onclick={startEdit} class="btn bg-base-100 btn-sm" title="Edit this deck">
					<Pencil class="w-4 h-4" />
				</button>
				<button
					onclick={() => (showDeleteModal = true)}
					class="btn bg-base-100 btn-sm text-error"
					title="Delete this deck">
					<Trash2 class="w-4 h-4" />
				</button>
				<a href={resolve(`/decks/${deck.id}`)} class="btn btn-primary btn-sm"> Open </a>
			</div>
		</div>
	</div>
</div>

<!-- Edit Modal -->
<Modal open={showEditModal} onClose={() => (showEditModal = false)} title="Edit Deck">
	<form
		method="POST"
		action="/decks?/update"
		use:enhance={() => {
			isSaving = true;
			return async ({ result, update }) => {
				isSaving = false;

				// Refresh the decks view first
				await update();

				if (result.type === 'success') {
					showEditModal = false;
					notifications.success('Deck updated successfully!');
				} else if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to update deck'));
				}
			};
		}}>
		<input type="hidden" name="id" value={deck.id} />

		<div class="space-y-4">
			<FormField
				label="Name"
				id="edit-deck-name-{deck.id}"
				name="name"
				placeholder="e.g., Krenko Goblins"
				bind:value={editName}
				helper="A descriptive name for this deck"
				required />

			<FormField
				label="Description"
				id="edit-deck-description-{deck.id}"
				name="description"
				helper="Optional description">
				<textarea
					id="edit-deck-description-{deck.id}"
					name="description"
					bind:value={editDescription}
					class="textarea textarea-bordered w-full"
					rows="3"
					placeholder="What is this deck about?"></textarea>
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
<Modal open={showDeleteModal} onClose={() => (showDeleteModal = false)} title="Delete Deck">
	<p>
		Are you sure you want to delete <span class="font-semibold">{deck.name}</span>? This will also
		remove all {deck.total_items} item{deck.total_items === 1 ? '' : 's'} in the deck. This action cannot
		be undone.
	</p>

	<form
		method="POST"
		action="/decks?/delete"
		use:enhance={() => {
			isDeleting = true;
			return async ({ result, update }) => {
				isDeleting = false;

				// Refresh the decks view first
				await update();

				if (result.type === 'success') {
					showDeleteModal = false;
					notifications.success('Deck deleted successfully!');
				} else if (result.type === 'failure') {
					notifications.error(getActionError(result.data, 'Failed to delete deck'));
				}
			};
		}}>
		<input type="hidden" name="id" value={deck.id} />

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
