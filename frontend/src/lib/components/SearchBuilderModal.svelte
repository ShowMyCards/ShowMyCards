<script lang="ts">
	import Modal from './Modal.svelte';
	import { buildSearchQuery } from '$lib/utils/search-builder';
	import type { Set as CardSet } from '$lib/types/models';

	interface Props {
		open: boolean;
		onClose: () => void;
		onSearch: (query: string) => void;
	}

	type CardSetResponse = {
		data: Pick<CardSet, 'code' | 'name'>[];
		total_pages: number;
	};

	let { open, onClose, onSearch }: Props = $props();

	let cardName = $state('');
	let cardType = $state('');
	let colors = $state(new Set<string>());
	let colorMode = $state<'all' | 'any' | 'exactly'>('all');

	let sets = $state<Pick<CardSet, 'code' | 'name'>[]>([]);
	let setsLoaded = $state(false);
	let setFilter = $state('');
	let selectedSet = $state<Pick<CardSet, 'code' | 'name'> | null>(null);
	let setDropdownOpen = $state(false);

	const NUMERIC_FIELDS = [
		{ id: 'mv', label: 'Mana Value', keyword: 'cmc' },
		{ id: 'pow', label: 'Power', keyword: 'pow' },
		{ id: 'loy', label: 'Loyalty', keyword: 'loy' },
		{ id: 'tou', label: 'Toughness', keyword: 'tou' }
	] as const;

	type NumericFieldId = (typeof NUMERIC_FIELDS)[number]['id'];
	type CompareOp = '=' | '>' | '<';

	let numericValues = $state<Record<NumericFieldId, string>>({ mv: '', pow: '', tou: '', loy: '' });
	let numericOps = $state<Record<NumericFieldId, CompareOp>>({
		mv: '=',
		pow: '=',
		tou: '=',
		loy: '='
	});

	const COLOR_OPTIONS = [
		{ value: 'w', label: 'White', symbol: 'https://svgs.scryfall.io/card-symbols/W.svg' },
		{ value: 'u', label: 'Blue', symbol: 'https://svgs.scryfall.io/card-symbols/U.svg' },
		{ value: 'b', label: 'Black', symbol: 'https://svgs.scryfall.io/card-symbols/B.svg' },
		{ value: 'r', label: 'Red', symbol: 'https://svgs.scryfall.io/card-symbols/R.svg' },
		{ value: 'g', label: 'Green', symbol: 'https://svgs.scryfall.io/card-symbols/G.svg' },
		{ value: 'c', label: 'Colorless', symbol: 'https://svgs.scryfall.io/card-symbols/C.svg' }
	];

	$effect(() => {
		if (open && !setsLoaded) {
			loadSets();
			loadSets();
		}
	});

	async function loadSets() {
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
			// silently ignore — user can still type a set code manually
		}
	}

	const filteredSets = $derived(
		setFilter.trim()
			? sets.filter(
					(s) =>
						s.name.toLowerCase().includes(setFilter.toLowerCase()) ||
						s.code.toLowerCase().includes(setFilter.toLowerCase())
				)
			: sets
	);

	function selectSet(s: Pick<CardSet, 'code' | 'name'>) {
		selectedSet = s;
		setFilter = '';
		setDropdownOpen = false;
	}

	function clearSet() {
		selectedSet = null;
		setFilter = '';
	}

	function handleSetBlur() {
		// Delay so that click on a list item registers before the list disappears
		setTimeout(() => {
			setDropdownOpen = false;
		}, 150);
	}

	function toggleColor(value: string) {
		const next = new Set(colors);
		if (next.has(value)) {
			next.delete(value);
		} else {
			next.add(value);
		}
		colors = next;
	}

	function buildQuery(): string {
		return buildSearchQuery({
			cardName,
			setCode: selectedSet?.code ?? setFilter,
			cardType,
			colors,
			colorMode,
			numericValues,
			numericOps
		});
	}

	function handleSearch() {
		const query = buildQuery();
		if (query) {
			onSearch(query);
			onClose();
		}
	}

	function handleClose() {
		cardName = '';
		cardType = '';
		colors = new Set();
		colorMode = 'all';
		selectedSet = null;
		setFilter = '';
		setDropdownOpen = false;
		for (const field of NUMERIC_FIELDS) {
			numericValues[field.id] = '';
			numericOps[field.id] = '=';
		}
		onClose();
	}
</script>

<Modal {open} onClose={handleClose} title="Advanced Search" boxClass="max-w-1/3">
	<div class="space-y-4">
		<div class="form-control">
			<label class="label" for="sb-name">
				<span class="label-text font-semibold">Card Name</span>
			</label>
			<input
				id="sb-name"
				type="text"
				placeholder="e.g. Lightning Bolt"
				bind:value={cardName}
				class="input input-bordered w-full" />
		</div>

		<div class="form-control">
			<label class="label" for="sb-set">
				<span class="label-text font-semibold">Set</span>
			</label>
			{#if selectedSet}
				<div class="input input-bordered flex items-center gap-2">
					<img
						src="/api/sets/code/{selectedSet.code}/icon"
						class="set-icon-img w-4 h-4 shrink-0"
						alt=""
						aria-hidden="true" />
					<span class="flex-1 truncate">{selectedSet.name}</span>
					<button
						type="button"
						class="btn btn-xs btn-ghost btn-circle shrink-0"
						aria-label="Clear set"
						onclick={clearSet}>✕</button>
				</div>
			{:else}
				<div class="relative">
					<input
						id="sb-set"
						type="text"
						placeholder={setsLoaded ? 'Search by name or code…' : 'Loading sets…'}
						bind:value={setFilter}
						onfocus={() => (setDropdownOpen = true)}
						onblur={handleSetBlur}
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
		</div>

		<div class="form-control">
			<label class="label" for="sb-type">
				<span class="label-text font-semibold">Card Type</span>
			</label>
			<input
				id="sb-type"
				type="text"
				placeholder="e.g. Creature, Instant, Legendary Creature"
				bind:value={cardType}
				class="input input-bordered w-full" />
		</div>

		<div class="form-control">
			<div class="label">
				<span class="label-text font-semibold">Colors</span>
			</div>
			<div class="flex flex-wrap gap-3">
				{#each COLOR_OPTIONS as { value, label, symbol } (value)}
					<label class="flex items-center gap-1 cursor-pointer" title={label}>
						<input
							type="checkbox"
							class="checkbox checkbox-primary checkbox-sm"
							checked={colors.has(value)}
							onchange={() => toggleColor(value)} />
						<img src={symbol} alt={label} class="w-6 h-6" />
					</label>
				{/each}
			</div>
		</div>

		{#if colors.size > 0}
			<div class="form-control">
				<label class="label" for="sb-color-mode">
					<span class="label-text font-semibold">Color Match</span>
				</label>
				<select id="sb-color-mode" bind:value={colorMode} class="select select-bordered w-full">
					<option value="all">All selected colors (and possibly more)</option>
					<option value="any">Any of the selected colors</option>
					<option value="exactly">Exactly these colors</option>
				</select>
			</div>
		{/if}

		<div class="grid grid-cols-2 gap-3">
			{#each NUMERIC_FIELDS as field (field.id)}
				<div class="form-control">
					<label class="label" for="sb-{field.id}">
						<span class="label-text font-semibold">{field.label}</span>
					</label>
					<div class="flex gap-1">
						<select
							bind:value={numericOps[field.id]}
							aria-label="{field.label} operator"
							class="select select-bordered w-auto">
							<option value="=">=</option>
							<option value=">">&gt;</option>
							<option value="<">&lt;</option>
						</select>
						<input
							id="sb-{field.id}"
							type="number"
							step="1"
							min="0"
							value={numericValues[field.id]}
							oninput={(e) => (numericValues[field.id] = e.currentTarget.value)}
							class="input input-bordered w-full min-w-0" />
					</div>
				</div>
			{/each}
		</div>
	</div>

	{#snippet actions()}
		<button type="button" class="btn btn-ghost" onclick={handleClose}>Cancel</button>
		<button type="button" class="btn btn-primary" onclick={handleSearch}>Search</button>
	{/snippet}
</Modal>

<style>
	:global([data-theme='dark']) .set-icon-img,
	:global([data-theme='halloween']) .set-icon-img,
	:global([data-theme='synthwave']) .set-icon-img,
	:global(.dark) .set-icon-img {
		filter: invert(1);
	}
</style>
