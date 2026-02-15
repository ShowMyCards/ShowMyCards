<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title: string;
		description?: string;
		actions?: Snippet;
	}

	let { title, description, actions }: Props = $props();
</script>

<div class="flex flex-col sm:flex-row sm:justify-between sm:items-center gap-4 mb-6">
	<div>
		<h1 class="text-3xl font-bold">{title}</h1>
		{#if description}
			<p class="text-sm opacity-70 mt-1">{description}</p>
		{/if}
	</div>
	{#if actions}
		<div class="flex gap-2">
			{@render actions()}
		</div>
	{/if}
</div>
