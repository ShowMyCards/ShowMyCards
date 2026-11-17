<script lang="ts">
	import { notifications } from '#lib';
	import {
		formatExportLines,
		type CardListExport,
		type CardListExportMode
	} from '#lib/utils/card-list-export.ts';
	import { copyText } from '#lib/utils/clipboard.ts';
	import { Copy } from '@lucide/svelte';
	import Modal from './Modal.svelte';

	interface Props {
		open: boolean;
		onClose: () => void;
		/** Every wanted card. */
		all: CardListExport;
		/** Only the cards still to get (not owned / not collected). */
		missing: CardListExport;
		title?: string;
	}

	let { open, onClose, all, missing, title = 'Export card list' }: Props = $props();

	const uid = $props.id();

	// Opens on the buying view every time; reset on close rather than persisted.
	let mode = $state<CardListExportMode>('missing');
	let textarea = $state<HTMLTextAreaElement>();

	const current = $derived(mode === 'all' ? all : missing);
	const text = $derived(formatExportLines(current.lines));
	const cardCount = $derived(current.lines.reduce((sum, line) => sum + line.quantity, 0));

	function close() {
		mode = 'missing';
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
		<div class="join" role="radiogroup" aria-label="Cards to export">
			<input
				type="radio"
				name="{uid}-mode"
				class="join-item btn btn-sm"
				aria-label="Only missing"
				value="missing"
				bind:group={mode} />
			<input
				type="radio"
				name="{uid}-mode"
				class="join-item btn btn-sm"
				aria-label="All cards"
				value="all"
				bind:group={mode} />
		</div>

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
				card{cardCount === 1 ? '' : 's'}. Formatted for Cardmarket Wants lists. Expansions
				Cardmarket names differently are added as any printing.
			</p>
		{:else}
			<div
				class="rounded-box border border-dashed border-base-300 p-6 text-center text-sm opacity-70">
				{mode === 'missing' ? 'Nothing missing: you have every card.' : 'No cards to export.'}
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
