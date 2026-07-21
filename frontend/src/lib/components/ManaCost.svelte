<script lang="ts">
	import { manaSymbolClasses } from '$lib/utils/mana';

	let { cost, class: className = '' }: { cost?: string | null; class?: string } = $props();

	const symbols = $derived(manaSymbolClasses(cost));
</script>

{#if symbols.length > 0}
	<span class="inline-flex items-center gap-0.5 {className}" role="img" aria-label={cost ?? ''}>
		{#each symbols as cls, i (i)}
			<i class={cls} aria-hidden="true"></i>
		{/each}
	</span>
{/if}
