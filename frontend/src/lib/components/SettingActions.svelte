<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		children: Snippet;
	}

	let { children }: Props = $props();
</script>

<div class="flex justify-end gap-2 pt-6 border-t border-base-300">
	{@render children()}
</div>
