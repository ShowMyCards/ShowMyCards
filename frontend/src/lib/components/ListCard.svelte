<script lang="ts">
	import type { ListSummary } from '$lib';
	import { resolve } from '$app/paths';
	import { ListTodo } from '@lucide/svelte';

	let { list }: { list: ListSummary } = $props();

	const progressColor = $derived.by(() => {
		if (list.completion_percentage >= 67) return 'progress-success';
		if (list.completion_percentage >= 34) return 'progress-warning';
		return 'progress-error';
	});
</script>

<div class="card bg-base-200 shadow-lg">
	<div class="card-body">
		<div class="flex items-start justify-between gap-4">
			<!-- Icon -->
			<div class="shrink-0 mt-1">
				<ListTodo class="w-6 h-6 text-primary" />
			</div>

			<!-- Content -->
			<div class="flex-1 min-w-0">
				<h3 class="font-semibold text-lg truncate">{list.name}</h3>
				{#if list.description}
					<p class="text-sm opacity-70 mt-1">{list.description}</p>
				{/if}

				<!-- Stats -->
				<div class="mt-3 space-y-2">
					<div class="text-sm">
						<span class="opacity-70"
							>{list.total_items} item{list.total_items === 1 ? '' : 's'}</span>
						<span class="opacity-50 mx-2">•</span>
						<span class="opacity-70">{list.completion_percentage}% complete</span>
					</div>

					<!-- Progress Bar -->
					<progress
						class="progress {progressColor} w-full"
						value={list.total_cards_collected}
						max={list.total_cards_wanted}></progress>

					<div class="text-xs opacity-60">
						{list.total_cards_collected} / {list.total_cards_wanted} cards collected
					</div>
				</div>
			</div>

			<!-- Browse Button -->
			<div class="shrink-0">
				<a href={resolve(`/lists/${list.id}`)} class="btn btn-primary btn-sm"> Browse </a>
			</div>
		</div>
	</div>
</div>
