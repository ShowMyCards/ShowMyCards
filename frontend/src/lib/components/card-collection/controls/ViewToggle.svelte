<script lang="ts">
	import { LayoutGrid, List } from '@lucide/svelte';
	import type { ViewMode } from '../types';

	interface Props {
		viewMode: ViewMode;
		onViewModeChange: (mode: ViewMode) => void;
	}

	let { viewMode, onViewModeChange }: Props = $props();
</script>

<div class="join">
	<button
		class="join-item btn btn-sm"
		class:btn-active={viewMode === 'grid'}
		onclick={() => onViewModeChange('grid')}
		title="Grid view">
		<LayoutGrid class="w-4 h-4" />
		<span class="sr-only">Grid view</span>
	</button>
	<button
		class="join-item btn btn-sm"
		class:btn-active={viewMode === 'table'}
		onclick={() => onViewModeChange('table')}
		title="Table view">
		<List class="w-4 h-4" />
		<span class="sr-only">Table view</span>
	</button>
</div>
