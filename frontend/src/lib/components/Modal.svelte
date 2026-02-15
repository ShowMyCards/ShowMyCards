<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		title?: string;
		children: Snippet;
		actions?: Snippet;
	}

	let { open, onClose, title, children, actions }: Props = $props();
</script>

{#if open}
	<div class="modal modal-open">
		<div class="modal-box max-w-2/3 max-h-2/3 w-full">
			{#if title}
				<h3 class="font-bold text-lg mb-4">{title}</h3>
			{/if}

			<div class="modal-content">
				{@render children()}
			</div>

			{#if actions}
				<div class="modal-action">
					{@render actions()}
				</div>
			{/if}
		</div>
		<button type="button" class="modal-backdrop" onclick={onClose} aria-label="Close modal"
		></button>
	</div>
{/if}
