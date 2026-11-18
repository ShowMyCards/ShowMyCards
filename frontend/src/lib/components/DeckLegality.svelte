<script lang="ts">
	import type { FormatLegality } from '#lib';
	import { Check, X, CircleAlert } from '@lucide/svelte';

	interface Props {
		/** Per-format verdicts in display order; null when the check failed. */
		formats: FormatLegality[] | null;
		/** True when the deck has no cards outside the maybe board. */
		empty: boolean;
	}

	let { formats, empty }: Props = $props();

	let openKey = $state<string | null>(null);

	// Groups in first-seen order; the backend sends formats in display order.
	const groups = $derived.by(() => {
		const result: { name: string; formats: FormatLegality[] }[] = [];
		for (const f of formats ?? []) {
			const group = result.find((g) => g.name === f.group);
			if (group) group.formats.push(f);
			else result.push({ name: f.group, formats: [f] });
		}
		return result;
	});

	// Derived rather than stored, so the reasons close by themselves when an edit
	// makes the open format legal.
	const open = $derived(formats?.find((f) => f.format === openKey && !f.legal) ?? null);

	function toggle(key: string) {
		openKey = open?.format === key ? null : key;
	}
</script>

<section class="mb-6" aria-labelledby="deck-legality-heading">
	<h2 id="deck-legality-heading" class="mb-2 text-sm font-semibold opacity-70">Format legality</h2>

	{#if empty}
		<p class="text-sm opacity-60">Add cards to check format legality.</p>
	{:else if !formats}
		<p role="alert" class="flex items-center gap-2 text-sm text-error">
			<CircleAlert class="w-4 h-4 shrink-0" aria-hidden="true" />
			Couldn't check format legality.
		</p>
	{:else}
		<div class="space-y-1.5">
			{#each groups as group (group.name)}
				<div class="flex flex-wrap items-center gap-1.5">
					<span class="w-full text-xs opacity-60 sm:w-24 sm:shrink-0">{group.name}</span>
					{#each group.formats as f (f.format)}
						{#if f.legal}
							<span class="badge badge-sm badge-soft badge-success gap-1">
								<Check class="w-3 h-3" aria-hidden="true" />
								{f.name} <span class="sr-only">(legal)</span>
							</span>
						{:else}
							<button
								type="button"
								class="badge badge-sm badge-soft badge-error gap-1 cursor-pointer"
								class:badge-outline={open?.format === f.format}
								aria-expanded={open?.format === f.format}
								onclick={() => toggle(f.format)}>
								<X class="w-3 h-3" aria-hidden="true" />
								{f.name} <span class="sr-only">(not legal)</span>
							</button>
						{/if}
					{/each}
				</div>
			{/each}
		</div>

		{#if open}
			<div class="mt-3 rounded-box bg-base-200 px-4 py-3 text-sm">
				<p class="mb-1 font-medium">Not legal in {open.name}</p>
				<ul class="list-disc space-y-0.5 pl-5">
					{#each open.reasons as reason (reason)}
						<li>{reason}</li>
					{/each}
				</ul>
			</div>
		{/if}
	{/if}
</section>
