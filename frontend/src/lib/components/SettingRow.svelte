<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		label: string;
		description?: string;
		children: Snippet;
	}

	let { label, description, children }: Props = $props();
</script>

<div class="flex flex-col md:flex-row md:items-start gap-4 py-4">
	<div class="flex-1">
		<div class="font-semibold">{label}</div>
		{#if description}
			<div class="text-sm opacity-70 mt-1">{description}</div>
		{/if}
	</div>
	<div class="md:w-80 md:flex md:justify-end">
		{@render children()}
	</div>
</div>
