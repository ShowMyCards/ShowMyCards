<script lang="ts">
	import type { PageData } from './$types';
	import {
		PageHeader,
		EmptyState,
		StorageLocationDropdown,
		notifications,
		getCardTreatmentName,
		type EnhancedCardResult,
		type Inventory
	} from '$lib';
	import {
		parseCardList,
		resolveTreatment,
		getTreatmentDisplayName,
		getTreatmentMarker,
		type ParsedCard
	} from '$lib/utils/card-list-parser';
	import { deserialize } from '$app/forms';
	import { FileText, Search, Plus, Check, AlertCircle, Loader2, X } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// Input state
	let inputText = $state('');
	let selectedStorageLocation = $state<number | 'auto'>('auto');

	const textareaPlaceholder = '4 e:who cn:1056\n2! !"Lightning Bolt"\n1!! e:cmr cn:361\nsol ring';

	// Preview state - combines parsing and searching
	interface PreviewCard {
		parsed: ParsedCard;
		status: 'searching' | 'ready' | 'error' | 'adding' | 'added';
		searchResult?: EnhancedCardResult;
		resolvedTreatment?: string;
		resolvedTreatmentName?: string;
		error?: string;
		addedInventory?: Inventory;
	}
	let previewCards = $state<PreviewCard[]>([]);
	let parseErrors = $state<ParsedCard[]>([]);
	let isPreviewing = $state(false);
	let isImporting = $state(false);

	async function searchCard(query: string): Promise<EnhancedCardResult | null> {
		const formData = new FormData();
		formData.append('query', query);

		const response = await fetch('?/searchCard', {
			method: 'POST',
			body: formData
		});

		const result = deserialize(await response.text());

		if (result.type === 'success' && result.data) {
			const searchData = result.data as { success: boolean; data: { data: EnhancedCardResult[] } };
			if (searchData.data?.data?.length > 0) {
				return searchData.data.data[0];
			}
		}

		return null;
	}

	async function handleParseAndPreview() {
		if (isPreviewing) return;

		// Parse the input
		const result = parseCardList(inputText);
		parseErrors = result.errors;

		if (result.cards.length === 0) {
			previewCards = [];
			return;
		}

		// Initialize preview cards with searching status
		previewCards = result.cards.map((parsed) => ({
			parsed,
			status: 'searching' as const
		}));

		isPreviewing = true;

		// Search each card
		for (let i = 0; i < previewCards.length; i++) {
			const card = previewCards[i];

			try {
				const searchResult = await searchCard(card.parsed.query);

				if (!searchResult) {
					previewCards[i] = {
						...card,
						status: 'error',
						error: 'Card not found'
					};
					continue;
				}

				// Resolve the treatment based on preference and available finishes
				const resolvedTreatment = resolveTreatment(card.parsed.treatment, searchResult.finishes);

				if (!resolvedTreatment) {
					const treatmentName = getTreatmentDisplayName(card.parsed.treatment);
					previewCards[i] = {
						...card,
						status: 'error',
						searchResult,
						error: `${treatmentName} not available`
					};
					continue;
				}

				// Get the proper display name using frame_effects
				const resolvedTreatmentName = getCardTreatmentName(
					searchResult.finishes,
					searchResult.frame_effects || [],
					resolvedTreatment
				);

				previewCards[i] = {
					...card,
					status: 'ready',
					searchResult,
					resolvedTreatment,
					resolvedTreatmentName
				};
			} catch (e) {
				previewCards[i] = {
					...card,
					status: 'error',
					error: e instanceof Error ? e.message : 'Search failed'
				};
			}

			// Force reactivity update
			previewCards = [...previewCards];

			// Small delay to avoid rate limiting
			await new Promise((resolve) => setTimeout(resolve, 100));
		}

		isPreviewing = false;
	}

	function handleClear() {
		inputText = '';
		previewCards = [];
		parseErrors = [];
	}

	async function addToInventory(
		card: EnhancedCardResult,
		quantity: number,
		treatment: string
	): Promise<Inventory | null> {
		const formData = new FormData();
		formData.append('scryfall_id', card.id);
		formData.append('oracle_id', card.oracle_id);
		formData.append('treatment', treatment);
		formData.append('quantity', quantity.toString());
		formData.append('storage_location_id', selectedStorageLocation.toString());

		const response = await fetch('?/addInventory', {
			method: 'POST',
			body: formData
		});

		const result = deserialize(await response.text());

		if (result.type === 'success' && result.data) {
			const actionData = result.data as { success: boolean; data: Inventory };
			return actionData.data;
		}

		return null;
	}

	async function handleImportAll() {
		if (isImporting) return;
		isImporting = true;

		let successCount = 0;
		let errorCount = 0;

		for (let i = 0; i < previewCards.length; i++) {
			const card = previewCards[i];
			if (card.status !== 'ready' || !card.searchResult || !card.resolvedTreatment) {
				continue;
			}

			// Update status to adding
			previewCards[i] = { ...card, status: 'adding' };
			previewCards = [...previewCards];

			try {
				const inventory = await addToInventory(
					card.searchResult,
					card.parsed.quantity,
					card.resolvedTreatment
				);

				if (inventory) {
					previewCards[i] = {
						...card,
						status: 'added',
						addedInventory: inventory
					};
					successCount++;
				} else {
					previewCards[i] = {
						...card,
						status: 'error',
						error: 'Failed to add to inventory'
					};
					errorCount++;
				}
			} catch (e) {
				previewCards[i] = {
					...card,
					status: 'error',
					error: e instanceof Error ? e.message : 'Unknown error'
				};
				errorCount++;
			}

			previewCards = [...previewCards];

			// Small delay between requests
			await new Promise((resolve) => setTimeout(resolve, 50));
		}

		isImporting = false;

		if (successCount > 0) {
			notifications.success(`Added ${successCount} card${successCount !== 1 ? 's' : ''} to inventory`);
		}
		if (errorCount > 0) {
			notifications.error(`Failed to import ${errorCount} card${errorCount !== 1 ? 's' : ''}`);
		}
	}

	// Derived stats
	const totalQuantity = $derived(
		previewCards
			.filter((c) => c.status === 'ready' || c.status === 'added')
			.reduce((sum, card) => sum + card.parsed.quantity, 0)
	);
	const readyCount = $derived(previewCards.filter((c) => c.status === 'ready').length);
	const addedCount = $derived(previewCards.filter((c) => c.status === 'added').length);
	const errorCount = $derived(previewCards.filter((c) => c.status === 'error').length);
	const searchingCount = $derived(previewCards.filter((c) => c.status === 'searching').length);
</script>

<PageHeader title="Bulk Import" description="Import cards using Scryfall queries" />

<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
	<!-- Input Section -->
	<div class="card bg-base-200 shadow-lg">
		<div class="card-body">
			<h2 class="card-title flex items-center gap-2">
				<FileText class="w-5 h-5" />
				Card List
			</h2>

			<p class="text-sm text-base-content/70 mb-2">
				Enter one card per line using Scryfall search syntax. Prefix with quantity and treatment markers.
			</p>

			<textarea
				bind:value={inputText}
				class="textarea textarea-bordered w-full h-64 font-mono text-sm"
				placeholder={textareaPlaceholder}
				disabled={isPreviewing}></textarea>

			<div class="flex flex-wrap gap-2 mt-4">
				<button
					class="btn btn-primary"
					onclick={handleParseAndPreview}
					disabled={!inputText.trim() || isPreviewing}>
					{#if isPreviewing}
						<Loader2 class="w-4 h-4 animate-spin" />
						Searching...
					{:else}
						<Search class="w-4 h-4" />
						Parse & Preview
					{/if}
				</button>
				<button class="btn btn-ghost" onclick={handleClear} disabled={!inputText.trim() || isPreviewing}>
					<X class="w-4 h-4" />
					Clear
				</button>
			</div>

			<!-- Storage location override -->
			{#if data.storageLocations.length > 0}
				<div class="mt-4 pt-4 border-t border-base-300">
					<p class="text-sm font-medium mb-2">Storage Location</p>
					<StorageLocationDropdown
						locations={data.storageLocations}
						selected={selectedStorageLocation}
						onchange={(v) => (selectedStorageLocation = v)} />
				</div>
			{/if}
		</div>
	</div>

	<!-- Results Section -->
	<div class="card bg-base-200 shadow-lg">
		<div class="card-body">
			<div class="flex items-center justify-between">
				<h2 class="card-title flex items-center gap-2">
					<Plus class="w-5 h-5" />
					Import Preview
				</h2>

				{#if previewCards.length > 0 && !isPreviewing}
					<div class="flex items-center gap-4">
						{#if readyCount > 0}
							<span class="text-sm text-base-content/70">
								{readyCount} ready ({totalQuantity} cards)
							</span>
							<button
								class="btn btn-primary btn-sm"
								onclick={handleImportAll}
								disabled={isImporting || readyCount === 0}>
								{#if isImporting}
									<Loader2 class="w-4 h-4 animate-spin" />
									Importing...
								{:else}
									<Plus class="w-4 h-4" />
									Import All
								{/if}
							</button>
						{/if}
					</div>
				{/if}
			</div>

			{#if previewCards.length === 0 && !isPreviewing}
				<EmptyState message="Enter card queries and click Parse & Preview" />
			{:else}
				<!-- Progress bar during preview/import -->
				{#if isPreviewing || isImporting}
					<div class="mb-4">
						<div class="flex justify-between text-sm mb-1">
							<span>{isPreviewing ? 'Searching cards...' : 'Importing...'}</span>
							<span>
								{#if isPreviewing}
									{previewCards.length - searchingCount} / {previewCards.length}
								{:else}
									{addedCount} / {readyCount + addedCount}
								{/if}
							</span>
						</div>
						<progress
							class="progress progress-primary w-full"
							value={isPreviewing ? previewCards.length - searchingCount : addedCount}
							max={isPreviewing ? previewCards.length : readyCount + addedCount}></progress>
					</div>
				{/if}

				<!-- Summary stats -->
				{#if !isPreviewing && (errorCount > 0 || addedCount > 0)}
					<div class="flex gap-2 mb-4">
						{#if readyCount > 0}
							<span class="badge badge-info">{readyCount} ready</span>
						{/if}
						{#if addedCount > 0}
							<span class="badge badge-success">{addedCount} added</span>
						{/if}
						{#if errorCount > 0}
							<span class="badge badge-error">{errorCount} errors</span>
						{/if}
					</div>
				{/if}

				<!-- Card list -->
				<div class="overflow-x-auto max-h-96">
					<table class="table table-sm">
						<thead class="sticky top-0 bg-base-200">
							<tr>
								<th>Qty</th>
								<th>Card</th>
								<th>Treatment</th>
								<th>Status</th>
							</tr>
						</thead>
						<tbody>
							{#each previewCards as card, index (index)}
								{@const treatmentMarker = getTreatmentMarker(card.parsed.treatment)}
								<tr class:opacity-50={card.status === 'added'}>
									<td class="font-mono">{card.parsed.quantity}{treatmentMarker}</td>
									<td>
										{#if card.searchResult}
											<div class="font-medium">{card.searchResult.name}</div>
											<div class="text-xs text-base-content/60">{card.searchResult.set_name}</div>
										{:else if card.status === 'searching'}
											<span class="text-base-content/50 text-xs font-mono">{card.parsed.query}</span>
										{:else}
											<span class="text-xs font-mono">{card.parsed.query}</span>
										{/if}
									</td>
									<td>
										{#if card.resolvedTreatmentName}
											{@const isFoil = card.resolvedTreatment !== 'nonfoil'}
											<span
												class="badge badge-sm whitespace-nowrap"
												class:badge-warning={isFoil}
												class:bg-gradient-to-r={isFoil}
												class:from-yellow-200={isFoil}
												class:to-amber-300={isFoil}
												class:text-amber-900={isFoil}
												class:border-0={isFoil}>
												{card.resolvedTreatmentName}
											</span>
										{:else if card.status === 'searching'}
											<span class="text-base-content/50">...</span>
										{:else if card.status === 'error'}
											<span class="badge badge-sm badge-ghost">
												{getTreatmentDisplayName(card.parsed.treatment)}
											</span>
										{/if}
									</td>
									<td>
										{#if card.status === 'searching'}
											<span class="flex items-center gap-1 text-info">
												<Loader2 class="w-3 h-3 animate-spin" />
											</span>
										{:else if card.status === 'ready'}
											<span class="text-success">
												<Check class="w-4 h-4" />
											</span>
										{:else if card.status === 'adding'}
											<span class="flex items-center gap-1 text-info">
												<Loader2 class="w-3 h-3 animate-spin" />
											</span>
										{:else if card.status === 'added'}
											<span class="flex items-center gap-1 text-success">
												<Check class="w-3 h-3" />
												{#if card.addedInventory?.storage_location}
													→ {card.addedInventory.storage_location.name}
												{/if}
											</span>
										{:else if card.status === 'error'}
											<span class="flex items-center gap-1 text-error" title={card.error}>
												<AlertCircle class="w-3 h-3" />
												{card.error}
											</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>

				<!-- Parse errors -->
				{#if parseErrors.length > 0}
					<div class="mt-4 pt-4 border-t border-base-300">
						<h3 class="font-semibold text-warning flex items-center gap-2 mb-2">
							<AlertCircle class="w-4 h-4" />
							{parseErrors.length} line{parseErrors.length !== 1 ? 's' : ''} could not be parsed
						</h3>
						<ul class="text-sm text-base-content/70 space-y-1">
							{#each parseErrors as error (error.lineNumber)}
								<li class="font-mono">Line {error.lineNumber}: {error.line}</li>
							{/each}
						</ul>
					</div>
				{/if}
			{/if}
		</div>
	</div>
</div>

<!-- Format help -->
<div class="card bg-base-200 shadow-lg mt-6">
	<div class="card-body">
		<h2 class="card-title">Format Guide</h2>

		<div class="grid grid-cols-1 md:grid-cols-2 gap-6">
			<div>
				<h3 class="font-semibold mb-2">Syntax</h3>
				<p class="text-sm text-base-content/70 mb-3">
					Each line follows the pattern: <code class="bg-base-300 px-1 rounded">[quantity][treatment] [scryfall query]</code>
				</p>

				<div class="overflow-x-auto">
					<table class="table table-sm">
						<thead>
							<tr>
								<th>Marker</th>
								<th>Treatment</th>
							</tr>
						</thead>
						<tbody>
							<tr>
								<td class="font-mono">4</td>
								<td>Regular (nonfoil, or first available)</td>
							</tr>
							<tr>
								<td class="font-mono">4!</td>
								<td>Foil (any foil variant)</td>
							</tr>
							<tr>
								<td class="font-mono">4!!</td>
								<td>Etched</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>

			<div>
				<h3 class="font-semibold mb-2">Examples</h3>
				<pre class="text-sm bg-base-300 p-3 rounded font-mono whitespace-pre-wrap"># Regular cards
4 !"Lightning Bolt"
2 e:2xm cn:117

# Foil cards (uses actual foil type)
4! e:who cn:1056
1! !"Sol Ring" e:cmr

# Etched cards
1!! e:cmr cn:361

# No quantity = 1 copy
!"Black Lotus"</pre>
			</div>
		</div>

		<div class="mt-4 pt-4 border-t border-base-300">
			<p class="text-sm text-base-content/70">
				Uses <a href="https://scryfall.com/docs/syntax" target="_blank" rel="noopener" class="link link-primary">Scryfall search syntax</a>.
				Common filters: <code class="bg-base-300 px-1 rounded">e:SET</code> (set),
				<code class="bg-base-300 px-1 rounded">cn:NUM</code> (collector number),
				<code class="bg-base-300 px-1 rounded">!"Exact Name"</code> (exact match).
			</p>
		</div>
	</div>
</div>
