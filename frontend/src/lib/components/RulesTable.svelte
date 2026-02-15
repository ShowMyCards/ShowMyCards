<script lang="ts">
	import { TableCard, Pagination, SortableRuleRow, type SortingRule } from '$lib';
	import type { DragDropState } from '@thisux/sveltednd';

	interface Props {
		rules: SortingRule[];
		highlightedRuleId: number | null;
		currentPage: number;
		totalPages: number;
		onEdit: (rule: SortingRule) => void;
		onDelete: (rule: SortingRule) => void;
		onToggleEnabled: (rule: SortingRule) => void;
		onDrop: (state: DragDropState<SortingRule>) => void;
		onPageChange: (page: number) => void;
	}

	let {
		rules,
		highlightedRuleId,
		currentPage,
		totalPages,
		onEdit,
		onDelete,
		onToggleEnabled,
		onDrop,
		onPageChange
	}: Props = $props();
</script>

<TableCard>
	<table class="table">
		<thead>
			<tr>
				<th>Priority</th>
				<th>Name</th>
				<th>Expression</th>
				<th>Storage Location</th>
				<th>Status</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each rules as rule, index (rule.id)}
				<SortableRuleRow
					{rule}
					{index}
					isHighlighted={highlightedRuleId === rule.id}
					onEdit={() => onEdit(rule)}
					onDelete={() => onDelete(rule)}
					onToggleEnabled={() => onToggleEnabled(rule)}
					{onDrop} />
			{/each}
		</tbody>
	</table>
</TableCard>

<!-- Pagination -->
{#if totalPages > 1}
	<div class="mt-4">
		<Pagination {currentPage} {totalPages} {onPageChange} />
	</div>
{/if}

<style>
	:global(.dragging) {
		opacity: 0.5;
		box-shadow:
			0 10px 15px -3px rgb(0 0 0 / 0.1),
			0 4px 6px -4px rgb(0 0 0 / 0.1);
	}

	:global(.drag-over) {
		background-color: hsl(var(--p) / 0.15);
		box-shadow: 0 0 0 2px hsl(var(--p) / 0.3);
		transform: scale(1.01);
	}
</style>
