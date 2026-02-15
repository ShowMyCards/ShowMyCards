<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		message: string;
		children?: Snippet;
	}

	let { message, children }: Props = $props();
</script>

<div class="text-center py-12">
	<p class="text-lg opacity-70 mb-4">{message}</p>
	{#if children}
		{@render children()}
	{/if}
</div>
