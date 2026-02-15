<script lang="ts">
	import type { StorageLocation } from '$lib';
	import { Box, BookOpen } from '@lucide/svelte';

	interface Props {
		locations: StorageLocation[];
		selected: number | 'auto';
		onchange: (value: number | 'auto') => void;
		compact?: boolean;
	}

	let { locations, selected, onchange, compact = false }: Props = $props();

	function handleChange(event: Event & { currentTarget: HTMLSelectElement }) {
		const value = event.currentTarget.value;
		onchange(value === 'auto' ? 'auto' : parseInt(value));
	}

	// Get icon for storage type
	function getIcon(type: string) {
		return type === 'Binder' ? BookOpen : Box;
	}
</script>

<select
	class="select select-bordered w-full"
	class:select-sm={compact}
	value={selected}
	onchange={handleChange}>
	<option value="auto">Auto (from rules)</option>
	{#each locations as location}
		<option value={location.id}>
			{location.storage_type === 'Binder' ? '📖' : '📦'} {location.name}
		</option>
	{/each}
</select>
