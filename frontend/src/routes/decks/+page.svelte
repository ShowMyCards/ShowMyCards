<script lang="ts">
	import { browser } from '$app/environment';
	import type { PageData } from './$types';
	import { enhance } from '$app/forms';
	import {
		PageHeader,
		StatsCard,
		EmptyState,
		Modal,
		FormField,
		DeckCard,
		notifications,
		getActionError
	} from '$lib';
	import { Plus } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

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

	// Summary stats
	const totalDecks = $derived(data.decks.length);
	const totalDemand = $derived(data.decks.reduce((sum, deck) => sum + deck.total_cards_demand, 0));
	const totalShortfall = $derived(
		data.decks.reduce((sum, deck) => sum + deck.aggregate_shortfall, 0)
	);
</script>

<svelte:head>
	<title>Decks - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader title="Decks" description="Build decks from the cards you own">
		{#snippet actions()}
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				<Plus class="w-4 h-4" />
				New Deck
			</button>
		{/snippet}
	</PageHeader>

	{#if data.decks.length === 0}
		<EmptyState message="No decks yet">
			<button onclick={() => (showCreateModal = true)} class="btn btn-primary">
				Create your first deck
			</button>
		</EmptyState>
	{:else}
		<StatsCard
			stats={[
				{
					title: 'Total Decks',
					value: totalDecks,
					description: 'Decks you are building'
				},
				{
					title: 'Cards in Decks',
					value: totalDemand,
					description: 'Total copies your decks call for'
				},
				{
					title: 'Cards Short',
					value: totalShortfall,
					description: "Copies you don't yet own"
				}
			]}
			class="mb-6 w-full" />

		<div class="space-y-4">
			{#each data.decks as deck (deck.id)}
				<DeckCard {deck} />
			{/each}
		</div>
	{/if}
</div>

<Modal open={showCreateModal} onClose={() => (showCreateModal = false)} title="Create Deck">
	<form
		method="POST"
		action="?/create"
		use:enhance={() => {
			return async ({ result, update }) => {
				// Update the page data first so the new deck appears
				await update();

				// Then show notifications based on result
				if (result.type === 'success') {
					notifications.success('Deck created successfully!');
					// Close modal and reset the form on success
					showCreateModal = false;
					createName = '';
					createDescription = '';
				} else if (result.type === 'failure') {
					const errorMsg = getActionError(result.data, 'Failed to create deck');
					notifications.error(errorMsg);
				}
			};
		}}>
		<div class="space-y-4">
			<FormField
				label="Name"
				id="create-deck-name"
				name="name"
				placeholder="e.g., Krenko Goblins"
				bind:value={createName}
				helper="A descriptive name for this deck"
				required />

			<FormField
				label="Description"
				id="create-deck-description"
				name="description"
				helper="Optional description">
				<textarea
					id="create-deck-description"
					name="description"
					bind:value={createDescription}
					class="textarea textarea-bordered w-full"
					rows="3"
					placeholder="What is this deck about?"></textarea>
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
