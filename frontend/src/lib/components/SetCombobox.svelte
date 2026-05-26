<script lang="ts">
	import type { Set as CardSet } from '$lib/types/models';

	type CardSetResponse = {
		data: Pick<CardSet, 'code' | 'name'>[];
		total_pages: number;
	};

	interface Props {
		id?: string;
		selected?: Pick<CardSet, 'code' | 'name'> | null;
	}

	let { id = 'sb-set', selected = $bindable(null) }: Props = $props();
	let inputText = $state('');

	let sets = $state<Pick<CardSet, 'code' | 'name'>[]>([]);
	let setsLoaded = $state(false);
	let setsLoading = $state(false);
	let setsLoadFailed = $state(false);
	let setDropdownOpen = $state(false);
	let wrapperEl: HTMLDivElement | undefined = $state();

	async function loadSets() {
		setsLoading = true;
		try {
			const first = await fetch('/api/sets?page_size=100&page=1');
			if (!first.ok) return;
			const response = (await first.json()) as CardSetResponse;
			const accumulated: Pick<CardSet, 'code' | 'name'>[] = response.data ?? [];
			const totalPages: number = response.total_pages ?? 1;

			if (totalPages > 1) {
				const pages = Array.from({ length: totalPages - 1 }, (_, i) =>
					fetch(`/api/sets?page_size=100&page=${i + 2}`)
						.then((r) => r.json())
						.then((d) => (d as CardSetResponse).data ?? [])
				);
				const rest = await Promise.all(pages);
				for (const page of rest) accumulated.push(...page);
			}

			sets = accumulated.sort((a, b) => a.name.localeCompare(b.name));
			setsLoaded = true;
		} catch {
			setsLoadFailed = true;
		} finally {
			setsLoading = false;
		}
	}

	const filteredSets = $derived(
		inputText.trim()
			? sets.filter(
					(s) =>
						s.name.toLowerCase().includes(inputText.toLowerCase()) ||
						s.code.toLowerCase().includes(inputText.toLowerCase())
				)
			: sets
	);

	function selectSet(s: Pick<CardSet, 'code' | 'name'>) {
		selected = s;
		inputText = '';
		setDropdownOpen = false;
	}

	function clearSet() {
		selected = null;
		inputText = '';
	}

	function handleFocus() {
		if (!setsLoaded) loadSets();
		setDropdownOpen = true;
	}

	function handleFocusOut(e: FocusEvent) {
		if (e.relatedTarget && wrapperEl?.contains(e.relatedTarget as Node)) return;
		setDropdownOpen = false;
		if (!selected) inputText = '';
	}
</script>

{#if selected}
	<div class="input input-bordered flex items-center gap-2">
		<img
			src="/api/sets/code/{selected.code}/icon"
			class="set-icon-img w-4 h-4 shrink-0"
			alt=""
			aria-hidden="true" />
		<span class="flex-1 truncate">{selected.name}</span>
		<button
			type="button"
			class="btn btn-xs btn-ghost btn-circle shrink-0"
			aria-label="Clear set"
			onclick={clearSet}>✕</button>
	</div>
{:else}
	<div class="relative" bind:this={wrapperEl} onfocusout={handleFocusOut}>
		<input
			{id}
			type="text"
			placeholder={setsLoadFailed
				? 'Set search unavailable'
				: setsLoading
					? 'Loading sets...'
					: 'Search by name or code...'}
			bind:value={inputText}
			onfocus={handleFocus}
			class="select select-bordered w-full" />
		{#if setDropdownOpen && filteredSets.length > 0}
			<ul
				class="absolute z-50 w-full mt-1 max-h-52 overflow-y-auto bg-base-100 border border-base-300 rounded-box shadow-lg">
				{#each filteredSets as s (s.code)}
					<li>
						<button
							type="button"
							class="flex items-center gap-2 w-full px-3 py-2 text-left hover:bg-base-200 text-sm"
							onclick={() => selectSet(s)}>
							<img
								src="/api/sets/code/{s.code}/icon"
								class="set-icon-img w-4 h-4 shrink-0"
								alt=""
								aria-hidden="true" />
							<span class="flex-1 truncate">{s.name}</span>
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>
{/if}

<style>
	:global([data-theme='dark']) .set-icon-img,
	:global([data-theme='halloween']) .set-icon-img,
	:global([data-theme='synthwave']) .set-icon-img,
	:global(.dark) .set-icon-img {
		filter: invert(1);
	}
</style>
