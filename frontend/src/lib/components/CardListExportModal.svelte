<script lang="ts">
	import { notifications } from '#lib';
	import { formatExportLines, type CardListExportChoice } from '#lib/utils/card-list-export.ts';
	import { copyText } from '#lib/utils/clipboard.ts';
	import { Copy } from '@lucide/svelte';
	import Modal from './Modal.svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		/** What to export, e.g. "Only missing" and "All cards". The first opens; one hides the switch. */
		choices: CardListExportChoice[];
		title?: string;
	}

	let { open, onClose, choices, title = 'Export card list' }: Props = $props();

	const uid = $props.id();

	// Opens on the first choice every time; reset on close rather than persisted.
	let selected = $state(0);
	let textarea = $state<HTMLTextAreaElement>();

	const choice = $derived(choices[selected] ?? choices[0]);
	const current = $derived(choice.export);
	const text = $derived(formatExportLines(current.lines));
	const cardCount = $derived(current.lines.reduce((sum, line) => sum + line.quantity, 0));
	const hasExpansions = $derived(current.lines.some((line) => line.expansion));

	function close() {
		selected = 0;
		onClose();
	}

	async function copy() {
		const result = await copyText(text, textarea);
		if (result === 'copied') {
			const n = current.lines.length;
			notifications.success(`Copied ${n} line${n === 1 ? '' : 's'} to the clipboard`);
		} else {
			notifications.info('Press Ctrl+C (⌘C on Mac) to copy the selected list');
		}
	}
</script>

<Modal {open} onClose={close} {title} boxClass="max-w-xl">
	<div class="space-y-4">
		{#if choices.length > 1}
			<div class="join" role="radiogroup" aria-label="Cards to export">
				{#each choices as option, index (option.label)}
					<input
						type="radio"
						name="{uid}-choice"
						class="join-item btn btn-sm"
						aria-label={option.label}
						value={index}
						bind:group={selected} />
				{/each}
			</div>
		{/if}

		{#if current.lines.length > 0}
			<textarea
				bind:this={textarea}
				readonly
				aria-label="Exported card list"
				class="textarea textarea-bordered w-full font-mono text-sm"
				rows={Math.min(14, Math.max(4, current.lines.length))}
				value={text}></textarea>
			<p class="text-xs opacity-70">
				{current.lines.length} line{current.lines.length === 1 ? '' : 's'} · {cardCount}
				card{cardCount === 1 ? '' : 's'}. Formatted for Cardmarket Wants lists.
				{#if hasExpansions}
					Expansions Cardmarket names differently are added as any printing.
				{/if}
			</p>
		{:else}
			<div
				class="rounded-box border border-dashed border-base-300 p-6 text-center text-sm opacity-70">
				{choice.empty ?? 'No cards to export.'}
			</div>
		{/if}

		{#if current.skipped > 0}
			<p class="text-xs text-warning">
				{current.skipped} item{current.skipped === 1 ? ' was' : 's were'} left out because
				{current.skipped === 1 ? 'its' : 'their'} card data couldn't be loaded.
			</p>
		{/if}
	</div>

	{#snippet actions()}
		<button type="button" class="btn btn-ghost" onclick={close}>Close</button>
		<button
			type="button"
			class="btn btn-primary"
			onclick={copy}
			disabled={current.lines.length === 0}>
			<Copy class="h-4 w-4" />
			Copy
		</button>
	{/snippet}
</Modal>
