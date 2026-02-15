<script lang="ts">
	import { browser } from '$app/environment';
	import { PageHeader, StatsCard, notifications } from '$lib';
	import type { PageData } from './$types';
	import { Search, Archive, Package, ClipboardList, Clock, Settings, History } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// Display load error via notifications store (browser only)
	let hasShownLoadError = false;
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	const stats = $derived([
		{
			title: 'Total Cards',
			value: (data?.stats?.total_inventory_cards ?? 0).toLocaleString(),
			description: 'In your collection',
			valueClass: 'text-primary'
		},
		{
			title: 'Collection Value',
			value: `$${(data?.stats?.total_collection_value ?? 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
			description: 'Total inventory value',
			valueClass: 'text-success'
		},
		{
			title: 'Storage Locations',
			value: (data?.stats?.total_storage_locations ?? 0).toLocaleString(),
			description: 'Boxes and binders',
			valueClass: 'text-secondary'
		},
		{
			title: 'Wishlist Cards',
			value: (data?.stats?.total_wishlist_cards ?? 0).toLocaleString(),
			description: 'Collected from lists',
			valueClass: 'text-accent'
		},
		{
			title: 'Collected Value',
			value: `$${(data?.stats?.total_collected_from_lists ?? 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
			description: 'From wishlists',
			valueClass: 'text-info'
		},
		{
			title: 'Remaining Value',
			value: `$${(data?.stats?.total_remaining_lists_value ?? 0).toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 })}`,
			description: 'Still needed',
			valueClass: 'text-warning'
		},
		{
			title: 'Unassigned Cards',
			value: (data?.stats?.unassigned_cards ?? 0).toLocaleString(),
			description: 'Need storage location',
			valueClass: data?.stats?.unassigned_cards ? 'text-error' : 'text-base-content'
		}
	]);
</script>

<svelte:head>
	<title>ShowMyCards - Dashboard</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader
		title="Dashboard"
		description="Welcome to ShowMyCards - Your card collection manager" />

	<div class="mb-6">
		<StatsCard {stats} />
	</div>

	<div class="grid grid-cols-1 md:grid-cols-3 gap-6">
		<a href="/search" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<Search class="h-6 w-6" />
					Search Cards
				</h2>
				<p class="text-sm opacity-70">Find and add cards to your collection</p>
			</div>
		</a>

		<a href="/inventory" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<Archive class="h-6 w-6" />
					Inventory
				</h2>
				<p class="text-sm opacity-70">Browse and manage your cards</p>
			</div>
		</a>

		<a href="/inventory/recent" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<History class="h-6 w-6" />
					Recently Added
				</h2>
				<p class="text-sm opacity-70">Quick access to recent cards</p>
			</div>
		</a>

		<a href="/storage" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<Package class="h-6 w-6" />
					Storage
				</h2>
				<p class="text-sm opacity-70">Organize boxes and binders</p>
			</div>
		</a>

		<a href="/rules" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<ClipboardList class="h-6 w-6" />
					Sorting Rules
				</h2>
				<p class="text-sm opacity-70">Automate card organization</p>
			</div>
		</a>

		<a href="/jobs" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<Clock class="h-6 w-6" />
					Jobs
				</h2>
				<p class="text-sm opacity-70">View scheduled tasks and history</p>
			</div>
		</a>

		<a href="/settings" class="card bg-base-200 shadow-lg hover:shadow-xl transition-shadow">
			<div class="card-body">
				<h2 class="card-title">
					<Settings class="h-6 w-6" />
					Settings
				</h2>
				<p class="text-sm opacity-70">Configure app preferences</p>
			</div>
		</a>
	</div>
</div>
