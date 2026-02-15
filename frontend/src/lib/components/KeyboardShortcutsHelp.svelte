<script lang="ts">
	import { keyboard } from '$lib';
	import { X, Keyboard } from '@lucide/svelte';

	const shortcuts = keyboard.getShortcuts();

	function handleBackdropKeydown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ') {
			keyboard.helpVisible = false;
		}
	}
</script>

{#if keyboard.helpVisible}
	<!-- svelte-ignore a11y_no_static_element_interactions -->
	<div
		class="fixed inset-0 bg-black/50 z-50 flex items-center justify-center"
		onclick={() => (keyboard.helpVisible = false)}
		onkeydown={handleBackdropKeydown}>
		<div class="bg-base-100 rounded-lg shadow-xl max-w-lg w-full mx-4 p-6" role="dialog" aria-label="Keyboard Shortcuts" tabindex="-1" onclick={(e) => e.stopPropagation()} onkeydown={(e) => e.stopPropagation()}>
			<div class="flex items-center justify-between mb-6">
				<h2 class="text-xl font-bold flex items-center gap-2">
					<Keyboard class="w-6 h-6" />
					Keyboard Shortcuts
				</h2>
				<button class="btn btn-sm btn-ghost btn-square" onclick={() => (keyboard.helpVisible = false)}>
					<X class="w-5 h-5" />
				</button>
			</div>

			<div class="space-y-2">
				{#each shortcuts as shortcut (shortcut.key)}
					<div class="flex items-center justify-between">
						<span class="text-base-content/80">{shortcut.description}</span>
						<kbd class="kbd kbd-sm">{shortcut.key}</kbd>
					</div>
				{/each}
			</div>

			<div class="mt-6 pt-4 border-t border-base-300">
				<p class="text-sm text-base-content/60">
					Hover over a card and use <kbd class="kbd kbd-xs">Alt+N</kbd> to add or <kbd class="kbd kbd-xs">Alt+Shift+N</kbd> to remove,
					where N is the treatment number shown on the card.
				</p>
			</div>
		</div>
	</div>
{/if}
