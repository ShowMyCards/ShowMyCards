<script lang="ts">
	import type { PageData } from './$types';
	import {
		PageHeader,
		EmptyState,
		StorageLocationDropdown,
		Modal,
		FormField,
		notifications,
		getCardTreatmentName,
		getActionError,
		SCRYFALL_LANGUAGES,
		type EnhancedCardResult,
		type CardResult,
		type ResolveResult,
		type Inventory
	} from '$lib';
	import {
		parseCardList,
		resolveTreatment,
		getTreatmentDisplayName,
		getTreatmentMarker,
		type ParsedCard,
		type ImportZone
	} from '$lib/utils/card-list-parser';
	import { deserialize } from '$app/forms';
	import { invalidateAll } from '$app/navigation';
	import {
		FileText,
		Search,
		Plus,
		Check,
		AlertCircle,
		Loader2,
		X,
		Package,
		Layers
	} from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// Destination: inventory (default) or a deck.
	let destination = $state<'inventory' | 'deck'>('inventory');
	let selectedDeckId = $state('');

	// Inline "create new deck" modal.
	let showCreateDeckModal = $state(false);
	let newDeckName = $state('');
	let newDeckDescription = $state('');
	let creatingDeck = $state(false);

	// Input state
	let inputText = $state('');
	let selectedLanguage = $state('en');
	let selectedStorageLocation = $state<number | 'auto'>('auto');

	const textareaPlaceholder =
		"1 Ashnod's Altar (BRR) 67 *F*\n4 Lightning Bolt\n\nSideboard\n2 Damn";

	const ZONE_LABELS: Record<ImportZone, string> = {
		command: 'Command',
		main: 'Main',
		side: 'Side',
		maybe: 'Maybe'
	};

	// Preview state - combines parsing and searching
	interface PreviewCard {
		parsed: ParsedCard;
		status: 'searching' | 'ready' | 'error' | 'adding' | 'added';
		searchResult?: CardResult;
		resolvedTreatment?: string;
		resolvedTreatmentName?: string;
		error?: string;
		addedInventory?: Inventory;
	}
	let previewCards = $state<PreviewCard[]>([]);
	let parseErrors = $state<ParsedCard[]>([]);
	let isPreviewing = $state(false);
	let isImporting = $state(false);

	const canImportToDeck = $derived(destination === 'deck' && selectedDeckId !== '');

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

	// Resolve the treatment for a found card and set the preview row's final state.
	function finalizePreviewCard(index: number, searchResult: CardResult | null) {
		const card = previewCards[index];
		if (!searchResult) {
			previewCards[index] = { ...card, status: 'error', error: 'Card not found' };
			return;
		}

		const resolvedTreatment = resolveTreatment(card.parsed.treatment, searchResult.finishes);
		if (!resolvedTreatment) {
			previewCards[index] = {
				...card,
				status: 'error',
				searchResult,
				error: `${getTreatmentDisplayName(card.parsed.treatment)} not available`
			};
			return;
		}

		const resolvedTreatmentName = getCardTreatmentName(
			searchResult.finishes,
			searchResult.frame_effects || [],
			resolvedTreatment
		);
		previewCards[index] = {
			...card,
			status: 'ready',
			searchResult,
			resolvedTreatment,
			resolvedTreatmentName
		};
	}

	async function resolveLocalBatch(
		items: { set?: string; collector_number?: string; name?: string }[]
	): Promise<ResolveResult[] | null> {
		const formData = new FormData();
		formData.append('items', JSON.stringify(items));
		formData.append('language', selectedLanguage);

		const response = await fetch('?/resolveLocal', { method: 'POST', body: formData });
		const result = deserialize(await response.text());

		if (result.type === 'success' && result.data) {
			const payload = result.data as { data?: { results?: ResolveResult[] } };
			return payload.data?.results ?? null;
		}
		return null;
	}

	async function handleParseAndPreview() {
		if (isPreviewing) return;

		// The native parser understands decklist exports (zones + pinned printings +
		// *F*/*E*) and Scryfall syntax, and appends the chosen language clause.
		const result = parseCardList(inputText, { language: selectedLanguage });
		parseErrors = result.errors;

		if (result.cards.length === 0) {
			previewCards = [];
			return;
		}

		previewCards = result.cards.map((parsed) => ({ parsed, status: 'searching' as const }));
		isPreviewing = true;

		// 1) Resolve pinned (set+collector) and plain-name lines against the local DB
		//    in one batch — fast, and free of Scryfall's outbound rate limit. Raw
		//    Scryfall-query lines (no parsed name) skip local resolution.
		const localTargets: number[] = [];
		const resolveItems: { set?: string; collector_number?: string; name?: string }[] = [];
		for (let i = 0; i < previewCards.length; i++) {
			const p = previewCards[i].parsed;
			if (p.pinned && p.set && p.collectorNumber) {
				localTargets.push(i);
				resolveItems.push({ set: p.set, collector_number: p.collectorNumber });
			} else if (!p.pinned && p.name) {
				localTargets.push(i);
				resolveItems.push({ name: p.name });
			}
		}

		// Lines that must hit Scryfall: those not sent to local resolution, plus any
		// local misses discovered below.
		const localSet = new Set(localTargets);
		const fallback: number[] = [];
		for (let i = 0; i < previewCards.length; i++) {
			if (!localSet.has(i)) fallback.push(i);
		}

		if (resolveItems.length > 0) {
			const results = await resolveLocalBatch(resolveItems);
			for (let j = 0; j < localTargets.length; j++) {
				const index = localTargets[j];
				const resolved = results?.[j];
				if (resolved?.found && resolved.card) {
					finalizePreviewCard(index, resolved.card);
				} else {
					// Missing locally (or the resolve call failed) — fall back to Scryfall.
					fallback.push(index);
				}
			}
			previewCards = [...previewCards];
		}

		// 2) Scryfall fallback for local misses + raw-query lines, client-paced to
		//    respect the upstream rate limit.
		fallback.sort((a, b) => a - b);
		for (const index of fallback) {
			try {
				finalizePreviewCard(index, await searchCard(previewCards[index].parsed.query));
			} catch (e) {
				previewCards[index] = {
					...previewCards[index],
					status: 'error',
					error: e instanceof Error ? e.message : 'Search failed'
				};
			}
			previewCards = [...previewCards];
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
		card: CardResult,
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

	async function handleImportToInventory() {
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
			notifications.success(
				`Added ${successCount} card${successCount !== 1 ? 's' : ''} to inventory`
			);
		}
		if (errorCount > 0) {
			notifications.error(`Failed to import ${errorCount} card${errorCount !== 1 ? 's' : ''}`);
		}
	}

	async function handleImportToDeck() {
		if (isImporting || !canImportToDeck) return;

		const ready = previewCards.filter((c) => c.status === 'ready' && c.searchResult);
		if (ready.length === 0) return;

		isImporting = true;

		// Pinned lines lock a specific printing; plain lines keep only the Oracle ID.
		// Finish is carried through but is not part of the M1 availability math.
		const items = ready.map((c) => ({
			oracle_id: c.searchResult!.oracle_id,
			scryfall_id: c.parsed.pinned ? c.searchResult!.id : '',
			treatment: c.parsed.treatment === 'nonfoil' ? '' : c.parsed.treatment,
			desired_quantity: c.parsed.quantity,
			zone: c.parsed.zone
		}));

		const formData = new FormData();
		formData.append('deck_id', selectedDeckId);
		formData.append('items', JSON.stringify(items));

		try {
			const response = await fetch('?/addDeckItems', { method: 'POST', body: formData });
			const result = deserialize(await response.text());

			if (result.type === 'success') {
				previewCards = previewCards.map((c) =>
					c.status === 'ready' ? { ...c, status: 'added' } : c
				);
				const deckName = data.decks.find((d) => String(d.id) === selectedDeckId)?.name ?? 'deck';
				notifications.success(
					`Added ${items.length} card${items.length !== 1 ? 's' : ''} to ${deckName}`
				);
				await invalidateAll();
			} else {
				notifications.error(
					result.type === 'failure'
						? getActionError(result.data, 'Failed to add cards to deck')
						: 'Failed to add cards to deck'
				);
			}
		} catch (e) {
			notifications.error(e instanceof Error ? e.message : 'Failed to add cards to deck');
		} finally {
			isImporting = false;
		}
	}

	function handleImport() {
		if (destination === 'deck') {
			handleImportToDeck();
		} else {
			handleImportToInventory();
		}
	}

	async function handleCreateDeck() {
		if (creatingDeck || !newDeckName.trim()) return;
		creatingDeck = true;

		const formData = new FormData();
		formData.append('name', newDeckName.trim());
		formData.append('description', newDeckDescription.trim());

		try {
			const response = await fetch('?/createDeck', { method: 'POST', body: formData });
			const result = deserialize(await response.text());

			if (result.type === 'success' && result.data) {
				const created = (result.data as { deck: { id: number } }).deck;
				await invalidateAll();
				selectedDeckId = String(created.id);
				showCreateDeckModal = false;
				newDeckName = '';
				newDeckDescription = '';
				notifications.success('Deck created');
			} else {
				notifications.error(
					result.type === 'failure'
						? getActionError(result.data, 'Failed to create deck')
						: 'Failed to create deck'
				);
			}
		} catch (e) {
			notifications.error(e instanceof Error ? e.message : 'Failed to create deck');
		} finally {
			creatingDeck = false;
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

<PageHeader title="Bulk Import" description="Import cards to your inventory or a deck" />

<div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
	<!-- Input Section -->
	<div class="card bg-base-200 shadow-lg">
		<div class="card-body">
			<h2 class="card-title flex items-center gap-2">
				<FileText class="w-5 h-5" />
				Card List
			</h2>

			<!-- Destination selector -->
			<div class="mb-2">
				<p class="text-sm font-medium mb-2">Destination</p>
				<div class="join">
					<button
						class="btn join-item btn-sm"
						class:btn-primary={destination === 'inventory'}
						onclick={() => (destination = 'inventory')}>
						<Package class="w-4 h-4" />
						Inventory
					</button>
					<button
						class="btn join-item btn-sm"
						class:btn-primary={destination === 'deck'}
						onclick={() => (destination = 'deck')}>
						<Layers class="w-4 h-4" />
						Deck
					</button>
				</div>

				{#if destination === 'deck'}
					<div class="flex flex-wrap items-end gap-2 mt-3">
						<label class="flex flex-col gap-1 text-sm">
							<span class="font-medium">Target deck</span>
							<select bind:value={selectedDeckId} class="select select-bordered select-sm min-w-56">
								<option value="" disabled>Select a deck…</option>
								{#each data.decks as deck (deck.id)}
									<option value={String(deck.id)}>{deck.name}</option>
								{/each}
							</select>
						</label>
						<button class="btn btn-sm bg-base-100" onclick={() => (showCreateDeckModal = true)}>
							<Plus class="w-4 h-4" />
							New deck
						</button>
					</div>
					{#if data.decks.length === 0}
						<p class="text-xs text-base-content/60 mt-2">
							You have no decks yet — create one to import into.
						</p>
					{/if}
				{/if}
			</div>

			<p class="text-sm text-base-content/70 mb-2">
				Paste a decklist export (Moxfield, MTGGoldfish, MTGA/MTGO) or enter Scryfall search syntax.
				Section headers and blank-line blocks set zones for deck import.
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
				<button
					class="btn btn-ghost"
					onclick={handleClear}
					disabled={!inputText.trim() || isPreviewing}>
					<X class="w-4 h-4" />
					Clear
				</button>
			</div>

			<!-- Card language -->
			<div class="mt-4 pt-4 border-t border-base-300">
				<label for="card-language" class="text-sm font-medium mb-2 block">Card Language</label>
				<p id="card-language-help" class="text-xs text-base-content/60 mb-2">
					Applied to every line as <code class="bg-base-300 px-1 rounded">l:&lt;code&gt;</code>
					unless the line already specifies a language. Choose "Any language" to leave lines untouched.
				</p>
				<select
					id="card-language"
					aria-describedby="card-language-help"
					bind:value={selectedLanguage}
					class="select select-bordered w-full max-w-xs">
					<option value="">Any language</option>
					{#each SCRYFALL_LANGUAGES as lang (lang.code)}
						<option value={lang.code}>{lang.name}</option>
					{/each}
				</select>
			</div>

			<!-- Storage location override (inventory only) -->
			{#if destination === 'inventory' && data.storageLocations.length > 0}
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
								onclick={handleImport}
								disabled={isImporting ||
									readyCount === 0 ||
									(destination === 'deck' && !canImportToDeck)}>
								{#if isImporting}
									<Loader2 class="w-4 h-4 animate-spin" />
									Importing...
								{:else}
									<Plus class="w-4 h-4" />
									{destination === 'deck' ? 'Import to Deck' : 'Import to Inventory'}
								{/if}
							</button>
						{/if}
					</div>
				{/if}
			</div>

			{#if destination === 'deck' && !canImportToDeck && previewCards.length > 0 && !isPreviewing}
				<p class="text-xs text-warning mt-1">Select a target deck to enable import.</p>
			{/if}

			{#if previewCards.length === 0 && !isPreviewing}
				<EmptyState message="Enter a card list and click Parse & Preview" />
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
								{#if destination === 'deck'}
									<th>Zone</th>
								{/if}
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
											<div class="text-xs text-base-content/60">
												{card.searchResult.set_name}
												{#if card.parsed.pinned}
													<span class="badge badge-outline badge-xs ml-1">pinned</span>
												{/if}
											</div>
										{:else if card.status === 'searching'}
											<span class="text-base-content/50 text-xs font-mono"
												>{card.parsed.query}</span>
										{:else}
											<span class="text-xs font-mono">{card.parsed.query}</span>
										{/if}
									</td>
									{#if destination === 'deck'}
										<td>
											<span class="badge badge-ghost badge-sm"
												>{ZONE_LABELS[card.parsed.zone]}</span>
										</td>
									{/if}
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
				<h3 class="font-semibold mb-2">Decklist exports</h3>
				<p class="text-sm text-base-content/70 mb-3">
					Paste exports from Moxfield, MTGGoldfish, MTGA or MTGO. Each line is
					<code class="bg-base-300 px-1 rounded">[qty] Name [(SET) COLLECTOR] [*F*/*E*]</code>. A
					<code class="bg-base-300 px-1 rounded">(SET) COLLECTOR</code>
					suffix pins that exact printing; <code class="bg-base-300 px-1 rounded">*F*</code> is
					foil,
					<code class="bg-base-300 px-1 rounded">*E*</code> is etched.
				</p>
				<pre class="text-sm bg-base-300 p-3 rounded font-mono whitespace-pre-wrap">1 Ashnod's Altar
1 Ashnod's Altar (BRR) 67 *F*
1 Gravecrawler (2X2) 438 *E*</pre>
			</div>

			<div>
				<h3 class="font-semibold mb-2">Zones (deck import)</h3>
				<p class="text-sm text-base-content/70 mb-3">
					Section headers — <span class="font-medium">Commander</span>,
					<span class="font-medium">Deck</span>/<span class="font-medium">Mainboard</span>,
					<span class="font-medium">Sideboard</span>, <span class="font-medium">Maybeboard</span>
					— set the zone for following lines. Exports without headers use blank lines: the first block
					is the main deck, later blocks are the sideboard. Re-zone anything (e.g. a commander) from the
					deck page.
				</p>
				<pre class="text-sm bg-base-300 p-3 rounded font-mono whitespace-pre-wrap">Commander
1 Teysa Karlov

Deck
1 Sol Ring</pre>
			</div>
		</div>

		<div class="mt-4 pt-4 border-t border-base-300">
			<h3 class="font-semibold mb-2">Scryfall syntax</h3>
			<p class="text-sm text-base-content/70">
				You can also use <a
					href="https://scryfall.com/docs/syntax"
					target="_blank"
					rel="noopener"
					class="link link-primary">Scryfall search syntax</a>
				with a leading quantity and treatment marker (<code class="bg-base-300 px-1 rounded"
					>!</code>
				foil,
				<code class="bg-base-300 px-1 rounded">!!</code> etched), e.g.
				<code class="bg-base-300 px-1 rounded">4! e:who cn:1056</code> or
				<code class="bg-base-300 px-1 rounded">2 e:2xm cn:117</code>.
			</p>
		</div>
	</div>
</div>

<!-- Create Deck Modal -->
<Modal open={showCreateDeckModal} onClose={() => (showCreateDeckModal = false)} title="Create Deck">
	<div class="space-y-4">
		<FormField
			label="Name"
			id="import-new-deck-name"
			name="name"
			placeholder="e.g., Teysa Triggers"
			bind:value={newDeckName}
			helper="A descriptive name for this deck"
			required />

		<FormField
			label="Description"
			id="import-new-deck-description"
			name="description"
			helper="Optional description">
			<textarea
				id="import-new-deck-description"
				name="description"
				bind:value={newDeckDescription}
				class="textarea textarea-bordered w-full"
				rows="3"></textarea>
		</FormField>
	</div>

	<div class="modal-action">
		<button
			type="button"
			onclick={() => (showCreateDeckModal = false)}
			disabled={creatingDeck}
			class="btn bg-base-100">
			Cancel
		</button>
		<button
			type="button"
			onclick={handleCreateDeck}
			disabled={creatingDeck || !newDeckName.trim()}
			class="btn btn-primary">
			{#if creatingDeck}
				<span class="loading loading-spinner loading-sm"></span>
				Creating...
			{:else}
				Create
			{/if}
		</button>
	</div>
</Modal>
