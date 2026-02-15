<script lang="ts">
	import { browser } from '$app/environment';
	import type { PageData, ActionData } from './$types';
	import { enhance } from '$app/forms';
	import {
		PageHeader,
		StatsCard,
		EmptyState,
		Modal,
		FormField,
		ListCard,
		notifications
	} from '$lib';
	import { Plus } from '@lucide/svelte';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let showCreateModal = $state(false);
	let createName = $state('');
	let createDescription = $state('');

	// Display load error if present (browser only)
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	// Calculate summary stats
	const totalLists = $derived(data.lists.length);
	const totalItems = $derived(data.lists.reduce((sum, list) => sum + list.total_items, 0));
	const averageCompletion = $derived.by(() => {
		if (totalLists === 0) return 0;
		const sum = data.lists.reduce((sum, list) => sum + list.completion_percentage, 0);
		return Math.round(sum / totalLists);
	});
</script>

<svelte:head>
	<title>Lists - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader title="Lists" description="Track cards you want to collect">
		{#snippet actions()}
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				<Plus class="w-4 h-4" />
				New List
			</button>
		{/snippet}
	</PageHeader>

	{#if data.lists.length === 0}
		<EmptyState message="No lists yet">
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				Create your first list
			</button>
		</EmptyState>
	{:else}
		<StatsCard
			stats={[
				{
					title: 'Total Lists',
					value: totalLists,
					description: 'Collections you are tracking'
				},
				{
					title: 'Total Items',
					value: totalItems,
					description: 'Cards across all lists'
				},
				{
					title: 'Average Progress',
					value: `${averageCompletion}%`,
					description: 'Overall completion'
				}
			]}
			class="mb-6 w-full" />

		<div class="space-y-4">
			{#each data.lists as list (list.id)}
				<ListCard {list} />
			{/each}
		</div>
	{/if}
</div>

<Modal open={showCreateModal} onClose={() => (showCreateModal = false)} title="Create List">
	<form
		method="POST"
		action="?/create"
		use:enhance={() => {
			return async ({ update }) => {
				// Don't close modal yet - let the redirect happen
				await update({ reset: false });
			};
		}}>
		<div class="space-y-4">
			<FormField
				label="Name"
				id="create-name"
				name="name"
				placeholder="e.g., Commander Staples"
				bind:value={createName}
				helper="A descriptive name for this list"
				required />

			<FormField
				label="Description"
				id="create-description"
				name="description"
				placeholder="What are you collecting?"
				bind:value={createDescription}
				helper="Optional description">
				{#snippet children()}
					<textarea
						id="create-description"
						name="description"
						bind:value={createDescription}
						class="textarea textarea-bordered w-full"
						rows="3"
						placeholder="What are you collecting?"></textarea>
				{/snippet}
			</FormField>
		</div>

		<div class="modal-action">
			<button type="button" onclick={() => (showCreateModal = false)} class="btn bg-base-100">
				Cancel
			</button>
			<button type="submit" class="btn btn-primary">Create</button>
		</div>
	</form>
</Modal>
