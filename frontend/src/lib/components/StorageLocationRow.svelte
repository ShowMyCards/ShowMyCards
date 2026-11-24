<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import type { StorageLocation } from '#lib';
	import { notifications, getActionError } from '#lib';
	import {
		Box,
		BookOpen,
		Pencil,
		Trash2,
		Grid2x2,
		GripVertical,
		ChevronUp,
		ChevronDown
	} from '@lucide/svelte';
	import { draggable, droppable, type DragDropState } from '@thisux/sveltednd';

	interface Props {
		location: StorageLocation;
		index: number;
		total: number;
		onDrop: (state: DragDropState<StorageLocation>) => void;
		onMoveUp: () => void;
		onMoveDown: () => void;
	}

	let { location, index, total, onDrop, onMoveUp, onMoveDown }: Props = $props();

	// Edit state
	let isEditing = $state(false);
	let editName = $state('');
	let editType = $state<'Box' | 'Binder'>('Box');
	let isSubmitting = $state(false);

	// Start editing mode
	function startEdit() {
		isEditing = true;
		editName = location.name;
		editType = location.storage_type as 'Box' | 'Binder';
	}

	// Cancel editing
	function cancelEdit() {
		isEditing = false;
		editName = '';
		editType = 'Box';
	}
</script>

<div
	use:droppable={{
		container: index.toString(),
		callbacks: { onDrop }
	}}
	class="flex items-center gap-4 p-4 transition-all duration-200">
	{#if isEditing}
		<!-- Edit Mode -->
		<form
			method="POST"
			action="?/update"
			use:enhance={() => {
				isSubmitting = true;
				return async ({ result, update }) => {
					isSubmitting = false;

					// Update page data first
					await update();

					// Then handle result
					if (result.type === 'success') {
						isEditing = false;
						notifications.success('Storage location updated successfully!');
					} else if (result.type === 'failure') {
						const errorMsg = getActionError(result.data, 'Failed to update storage location');
						notifications.error(errorMsg);
					}
				};
			}}
			class="flex items-center gap-4 flex-1">
			<input type="hidden" name="id" value={location.id} />

			<!-- Icon Preview -->
			<div class="shrink-0">
				{#if editType === 'Box'}
					<Box class="w-6 h-6 text-primary" />
				{:else}
					<BookOpen class="w-6 h-6 text-secondary" />
				{/if}
			</div>

			<!-- Form Fields -->
			<div class="flex gap-3 flex-1">
				<input
					type="text"
					id="edit-name-{location.id}"
					name="name"
					bind:value={editName}
					class="input input-bordered input-sm flex-1"
					placeholder="Name"
					required />
				<select
					id="edit-type-{location.id}"
					name="storage_type"
					bind:value={editType}
					class="select select-bordered select-sm">
					<option value="Box">Box</option>
					<option value="Binder">Binder</option>
				</select>
			</div>

			<!-- Action Buttons -->
			<div class="flex gap-2 ml-auto">
				<button
					type="submit"
					disabled={isSubmitting}
					class="btn btn-primary btn-sm"
					title="Save changes">
					{#if isSubmitting}
						<span class="loading loading-spinner loading-xs"></span>
					{/if}
					Save
				</button>
				<button
					type="button"
					onclick={cancelEdit}
					disabled={isSubmitting}
					class="btn bg-base-100 btn-sm"
					title="Cancel editing">
					Cancel
				</button>
			</div>
		</form>
	{:else}
		<!-- View Mode -->
		<!-- Drag Handle -->
		<div
			use:draggable={{
				container: index.toString(),
				dragData: location
			}}
			class="cursor-grab active:cursor-grabbing p-1 -m-1 rounded hover:bg-base-300 shrink-0"
			title="Drag to reorder">
			<GripVertical class="w-4 h-4 opacity-60" />
		</div>

		<!-- Storage Type Icon -->
		<div class="shrink-0">
			{#if location.storage_type === 'Box'}
				<Box class="w-6 h-6 text-primary" />
			{:else}
				<BookOpen class="w-6 h-6 text-secondary" />
			{/if}
		</div>

		<!-- Name and Type -->
		<div class="flex-1">
			<div class="font-semibold">{location.name}</div>
			<div class="text-sm opacity-70">{location.storage_type}</div>
		</div>

		<!-- Action Buttons -->
		<div class="flex gap-2 ml-auto">
			<button
				type="button"
				onclick={onMoveUp}
				disabled={index === 0}
				class="btn bg-base-100 btn-sm"
				aria-label="Move {location.name} up"
				title="Move up">
				<ChevronUp class="w-4 h-4" />
			</button>
			<button
				type="button"
				onclick={onMoveDown}
				disabled={index === total - 1}
				class="btn bg-base-100 btn-sm"
				aria-label="Move {location.name} down"
				title="Move down">
				<ChevronDown class="w-4 h-4" />
			</button>
			<a
				href={resolve(`inventory/${location.id}`)}
				class="btn bg-base-100 btn-sm"
				title="Browse cards in this location">
				<Grid2x2 class="w-4 h-4" />
			</a>
			<button onclick={startEdit} class="btn bg-base-100 btn-sm" title="Edit this location">
				<Pencil class="w-4 h-4" />
			</button>
			<form
				method="POST"
				action="?/delete"
				use:enhance={() => {
					return async ({ result, update }) => {
						// Update page data first
						await update();

						// Then show notification
						if (result.type === 'success') {
							notifications.success('Storage location deleted successfully!');
						} else if (result.type === 'failure') {
							const errorMsg = getActionError(result.data, 'Failed to delete storage location');
							notifications.error(errorMsg);
						}
					};
				}}
				class="inline">
				<input type="hidden" name="id" value={location.id} />
				<button
					type="submit"
					class="btn bg-base-100 btn-sm text-error"
					title="Delete this location">
					<Trash2 class="w-4 h-4" />
				</button>
			</form>
		</div>
	{/if}
</div>
