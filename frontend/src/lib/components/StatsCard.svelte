<script lang="ts">
	import type { Snippet } from 'svelte';

	interface StatItem {
		title: string;
		value: string | number;
		description?: string;
		figure?: Snippet;
		valueClass?: string;
	}

	interface Props {
		stats: StatItem[];
		vertical?: boolean;
		class?: string;
	}

	let { stats, vertical = false, class: className }: Props = $props();

	const layoutClass = $derived(vertical ? 'stats-vertical' : 'stats-vertical lg:stats-horizontal');
</script>

<div class="stats {layoutClass} shadow {className || ''}">
	{#each stats as stat, index (index)}
		<div class="stat">
			{#if stat.figure}
				<div class="stat-figure">
					{@render stat.figure()}
				</div>
			{/if}
			<div class="stat-title">{stat.title}</div>
			<div class="stat-value {stat.valueClass || ''}">{stat.value}</div>
			{#if stat.description}
				<div class="stat-desc">{stat.description}</div>
			{/if}
		</div>
	{/each}
</div>
