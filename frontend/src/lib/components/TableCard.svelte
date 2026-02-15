<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		title?: string;
		stats?: Snippet;
		children: Snippet;
		actions?: Snippet;
		class?: string;
	}

	let { title, stats, children, actions, class: className }: Props = $props();
</script>

<div class="card bg-base-200 shadow-lg {className || ''}">
	<div class="card-body">
		{#if title}
			<h2 class="card-title mb-4">{title}</h2>
		{/if}

		{#if stats}
			<div class="mb-6">
				{@render stats()}
			</div>
		{/if}

		<div class="overflow-x-auto">
			{@render children()}
		</div>

		{#if actions}
			<div class="card-actions justify-end mt-4">
				{@render actions()}
			</div>
		{/if}
	</div>
</div>
