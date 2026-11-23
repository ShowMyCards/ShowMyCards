<script lang="ts">
	import Lozenge from './Lozenge.svelte';
	import Modal from './Modal.svelte';
	import { FileArchive } from '@lucide/svelte';
	import {
		formatBytes,
		formatDuration,
		metadataEntries,
		parseJobMetadata,
		statusColor,
		statusText,
		type JobRecord
	} from '../utils/jobs';

	let {
		job,
		typeLabel,
		onClose
	}: {
		/** The job to show; the modal is open while this is set. */
		job: JobRecord | null;
		typeLabel: string;
		onClose: () => void;
	} = $props();

	const metadata = $derived(parseJobMetadata(job?.metadata));
	const entries = $derived(metadata ? metadataEntries(metadata) : []);

	// A job that produced a file records it as file_name (+ size_bytes).
	const output = $derived.by(() => {
		if (typeof metadata?.file_name !== 'string') return null;
		const size = typeof metadata.size_bytes === 'number' ? formatBytes(metadata.size_bytes) : null;
		return { fileName: metadata.file_name, size };
	});

	function formatTime(value?: string): string {
		return value ? new Date(value).toLocaleString() : '—';
	}
</script>

<Modal open={job !== null} {onClose} title={job ? `Job #${job.id}` : ''} boxClass="max-w-2xl">
	{#if job}
		<div class="space-y-5">
			<div class="flex flex-wrap items-center gap-2">
				<Lozenge style="outline">{typeLabel}</Lozenge>
				<Lozenge color={statusColor(job.status)}>{statusText(job.status)}</Lozenge>
			</div>

			<dl class="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1 text-sm">
				<dt class="opacity-70">Created</dt>
				<dd>{formatTime(job.created_at)}</dd>
				<dt class="opacity-70">Started</dt>
				<dd>{formatTime(job.started_at)}</dd>
				<dt class="opacity-70">Finished</dt>
				<dd>{formatTime(job.completed_at)}</dd>
				<dt class="opacity-70">Duration</dt>
				<dd>{formatDuration(job)}</dd>
			</dl>

			{#if output}
				<section aria-label="Output">
					<h3 class="font-semibold mb-2">Output</h3>
					<div class="flex items-center gap-2 rounded bg-base-200 p-3 text-sm">
						<FileArchive class="w-4 h-4 shrink-0" />
						<span class="font-mono break-all">{output.fileName}</span>
						{#if output.size}
							<span class="opacity-70 shrink-0">({output.size})</span>
						{/if}
					</div>
				</section>
			{/if}

			{#if job.error}
				<section aria-label="Error">
					<h3 class="font-semibold mb-2 text-error">Error</h3>
					<pre
						class="bg-base-200 p-3 rounded overflow-x-auto text-sm whitespace-pre-wrap">{job.error}</pre>
				</section>
			{/if}

			{#if entries.length > 0}
				<section aria-label="Details">
					<h3 class="font-semibold mb-2">Details</h3>
					<dl class="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-1 text-sm">
						{#each entries as entry (entry.key)}
							<dt class="opacity-70">{entry.label}</dt>
							<dd class="min-w-0">
								{#if entry.structured}
									<pre class="bg-base-200 p-2 rounded overflow-x-auto text-xs">{entry.value}</pre>
								{:else}
									<span class="break-all">{entry.value}</span>
								{/if}
							</dd>
						{/each}
					</dl>
				</section>
			{/if}
		</div>

		<div class="modal-action">
			<button type="button" class="btn btn-ghost" onclick={onClose}>Close</button>
		</div>
	{/if}
</Modal>
