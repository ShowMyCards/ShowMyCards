<script lang="ts">
	import Modal from './Modal.svelte';
	import SetCombobox from './SetCombobox.svelte';
	import { buildSearchQuery } from '$lib/utils/search-builder';
	import type { Set as CardSet } from '$lib/types/models';
	import { SvelteSet } from 'svelte/reactivity';

	interface Props {
		open: boolean;
		onClose: () => void;
		onSearch: (query: string) => void;
	}

	let { open, onClose, onSearch }: Props = $props();

	let cardName = $state('');
	let cardType = $state('');
	let colors = $state(new Set<string>());
	let colorMode = $state<'all' | 'any' | 'exactly'>('all');

	let selectedSet = $state<Pick<CardSet, 'code' | 'name'> | null>(null);

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

	function toggleColor(value: string) {
		const next = new SvelteSet(colors);
		if (next.has(value)) {
			next.delete(value);
		} else if (value === 'c') {
			next.clear();
			next.add('c');
		} else {
			next.delete('c');
			next.add(value);
		}
		colors = next;
	}

	function buildQuery(): string {
		return buildSearchQuery({
			cardName,
			setCode: selectedSet?.code ?? '',
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
			<SetCombobox id="sb-set" bind:selected={selectedSet} />
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
