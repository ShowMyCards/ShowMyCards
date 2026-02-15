<script lang="ts">
	import { browser } from '$app/environment';
	import { afterNavigate, invalidateAll } from '$app/navigation';
	import type { PageData } from './$types';
	import { PageHeader, Notification, EmptyState, ResortModal } from '$lib';
	import { Box, BookOpen, Grid2x2, Package, AlertCircle, RefreshCw } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	let showResortModal = $state(false);

	// Always reload data when navigating to this page
	afterNavigate(() => {
		if (browser) {
			invalidateAll();
		}
	});

	function handleResortComplete() {
		invalidateAll();
	}
</script>

<svelte:head>
	<title>Inventory Browser - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader
		title="Inventory Browser"
		description="Browse your card collection by storage location">
		{#snippet actions()}
			<button class="btn btn-outline btn-sm" onclick={() => (showResortModal = true)}>
				<RefreshCw class="w-4 h-4" />
				Re-sort All
			</button>
		{/snippet}
	</PageHeader>

	{#if data.error}
		<Notification type="error">
			{data.error}
		</Notification>
	{/if}

	{#if data.locations.length === 0 && data.unassignedCount === 0}
		<EmptyState message="No inventory found">
			<a href="/search" class="btn btn-primary">
				<Package class="w-4 h-4" />
				Search for Cards
			</a>
		</EmptyState>
	{:else}
		<div class="card bg-base-200 shadow-lg">
			<div class="card-body p-0">
				<div class="divide-y divide-base-300">
					<!-- Unassigned Cards (if any) -->
					{#if data.unassignedCount > 0}
						<div class="flex items-center gap-4 p-4">
							<!-- Icon -->
							<div class="flex-shrink-0">
								<AlertCircle class="w-6 h-6 text-warning" />
							</div>

							<!-- Name and Card Count -->
							<div class="flex-1">
								<div class="font-semibold">Unassigned Cards</div>
								<div class="text-sm opacity-70">
									{data.unassignedCount} card{data.unassignedCount === 1 ? '' : 's'}
								</div>
							</div>

							<!-- Browse Button -->
							<div class="ml-auto">
								<a href="/inventory/unassigned" class="btn btn-primary btn-sm">
									<Grid2x2 class="w-4 h-4" />
									Browse
								</a>
							</div>
						</div>
					{/if}

					<!-- Storage Locations -->
					{#each data.locations as location (location.id)}
						<div class="flex items-center gap-4 p-4">
							<!-- Storage Type Icon -->
							<div class="flex-shrink-0">
								{#if location.storage_type === 'Box'}
									<Box class="w-6 h-6 text-primary" />
								{:else}
									<BookOpen class="w-6 h-6 text-secondary" />
								{/if}
							</div>

							<!-- Name, Type, and Card Count -->
							<div class="flex-1">
								<div class="font-semibold">{location.name}</div>
								<div class="text-sm opacity-70">
									{location.storage_type} • {location.card_count} card{location.card_count === 1
										? ''
										: 's'}
								</div>
							</div>

							<!-- Browse Button -->
							<div class="ml-auto">
								<a href="/inventory/{location.id}" class="btn btn-primary btn-sm">
									<Grid2x2 class="w-4 h-4" />
									Browse
								</a>
							</div>
						</div>
					{/each}
				</div>
			</div>
		</div>
	{/if}

	<ResortModal
		open={showResortModal}
		onclose={() => (showResortModal = false)}
		onComplete={handleResortComplete} />
</div>
