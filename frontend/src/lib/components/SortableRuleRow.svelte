<script lang="ts">
	import { GripVertical, Pencil, Trash2 } from '@lucide/svelte';
	import { draggable, droppable, type DragDropState } from '@thisux/sveltednd';
	import type { SortingRule } from '$lib';

	interface Props {
		rule: SortingRule;
		index: number;
		isHighlighted: boolean;
		onEdit: () => void;
		onDelete: () => void;
		onToggleEnabled: () => void;
		onDrop: (state: DragDropState<SortingRule>) => void;
	}

	let { rule, index, isHighlighted, onEdit, onDelete, onToggleEnabled, onDrop }: Props = $props();
</script>

<tr
	use:droppable={{
		container: index.toString(),
		callbacks: { onDrop }
	}}
	data-rule-id={rule.id}
	class:bg-success={isHighlighted}
	class:bg-opacity-20={isHighlighted}
	class="hover:bg-base-200 transition-all duration-200">
	<td>
		<div class="flex items-center gap-2">
			<div
				use:draggable={{
					container: index.toString(),
					dragData: rule
				}}
				class="cursor-grab active:cursor-grabbing p-1 -m-1 rounded hover:bg-base-300">
				<GripVertical class="w-4 h-4" />
			</div>
			<span class="font-semibold">{rule.priority}</span>
		</div>
	</td>
	<td class="font-medium">{rule.name}</td>
	<td>
		<code class="text-xs">{rule.expression}</code>
	</td>
	<td>{rule.storage_location?.name || 'N/A'}</td>
	<td>
		<input
			type="checkbox"
			checked={rule.enabled}
			onchange={onToggleEnabled}
			class="toggle toggle-sm toggle-success" />
	</td>
	<td>
		<div class="flex gap-2">
			<button onclick={onEdit} class="btn bg-base-100 btn-sm" title="Edit rule">
				<Pencil class="w-4 h-4" />
			</button>
			<button onclick={onDelete} class="btn bg-base-100 btn-sm text-error" title="Delete rule">
				<Trash2 class="w-4 h-4" />
			</button>
		</div>
	</td>
</tr>
