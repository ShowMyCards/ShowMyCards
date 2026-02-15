<script lang="ts">
	import { Search, X } from '@lucide/svelte';

	interface Props {
		/** Current search text */
		searchText?: string;
		/** Callback when search text changes */
		onSearchChange?: (text: string) => void;
		/** Current collection status filter */
		statusFilter?: 'all' | 'collected' | 'uncollected';
		/** Callback when status filter changes */
		onStatusChange?: (status: 'all' | 'collected' | 'uncollected') => void;
		/** Show status filter buttons */
		showStatusFilter?: boolean;
		/** Placeholder text for search input */
		placeholder?: string;
	}

	let {
		searchText = '',
		onSearchChange,
		statusFilter = 'all',
		onStatusChange,
		showStatusFilter = true,
		placeholder = 'Filter by name, set, or treatment...'
	}: Props = $props();

	let inputValue = $state('');

	// Sync internal state with prop changes
	$effect(() => {
		inputValue = searchText;
	});

	function handleInput(event: Event & { currentTarget: HTMLInputElement }) {
		inputValue = event.currentTarget.value;
		onSearchChange?.(event.currentTarget.value);
	}

	function clearSearch() {
		inputValue = '';
		onSearchChange?.('');
	}

	function handleStatusClick(status: 'all' | 'collected' | 'uncollected') {
		onStatusChange?.(status);
	}
</script>

<div class="flex flex-col sm:flex-row gap-3 items-start sm:items-center">
	<!-- Search input -->
	<div class="relative flex-1 w-full sm:max-w-sm">
		<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-base-content/50" />
		<input
			type="text"
			value={inputValue}
			oninput={handleInput}
			{placeholder}
			class="input input-bordered input-sm w-full pl-9 pr-8" />
		{#if inputValue}
			<button
				type="button"
				onclick={clearSearch}
				class="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 hover:bg-base-300 rounded"
				title="Clear search">
				<X class="w-4 h-4 text-base-content/50" />
			</button>
		{/if}
	</div>

	<!-- Status filter buttons -->
	{#if showStatusFilter}
		<div class="join">
			<button
				type="button"
				class="join-item btn btn-sm"
				class:btn-active={statusFilter === 'all'}
				onclick={() => handleStatusClick('all')}>
				All
			</button>
			<button
				type="button"
				class="join-item btn btn-sm"
				class:btn-active={statusFilter === 'collected'}
				onclick={() => handleStatusClick('collected')}>
				Collected
			</button>
			<button
				type="button"
				class="join-item btn btn-sm"
				class:btn-active={statusFilter === 'uncollected'}
				onclick={() => handleStatusClick('uncollected')}>
				Need
			</button>
		</div>
	{/if}
</div>
