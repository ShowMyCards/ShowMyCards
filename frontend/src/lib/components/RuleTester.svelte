<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Search, Zap, AlertCircle, CheckCircle2, XCircle } from '@lucide/svelte';
	import {
		type SortingRule,
		type StorageLocation,
		getCardTreatmentName,
		getAvailableTreatments
	} from '$lib';
	import type { CardResult } from '$lib/types/api';

	// Full Scryfall card type with all fields for evaluation
	interface ScryfallCard extends CardResult {
		// Additional fields from Scryfall API
		image_uris?: {
			small?: string;
			normal?: string;
			large?: string;
		};
		rarity?: string;
		colors?: string[];
		cmc?: number;
		type_line?: string;
		set?: string;
		set_type?: string;
		frame_effects?: string[];
		// Allow other Scryfall fields with reasonable types
		[key: string]: unknown;
	}

	interface Props {
		rules: SortingRule[];
		onRuleMatch?: (ruleId: number) => void;
	}

	let { rules, onRuleMatch }: Props = $props();

	// Search state
	let searchQuery = $state('');
	let searchResults = $state<ScryfallCard[]>([]);
	let isSearching = $state(false);
	let searchTimeout: ReturnType<typeof setTimeout> | null = null;

	onDestroy(() => {
		if (searchTimeout) clearTimeout(searchTimeout);
	});

	// Selected card and evaluation state
	let selectedCard = $state<ScryfallCard | null>(null);
	let selectedTreatment = $state<string>('nonfoil');
	let matchedRule = $state<SortingRule | null>(null);
	let matchedLocation = $state<StorageLocation | null>(null);
	let isEvaluating = $state(false);
	let evaluationError = $state('');

	// Get available treatments with proper display names
	const availableTreatments = $derived.by(() => {
		if (!selectedCard || !selectedCard.finishes || selectedCard.finishes.length === 0) {
			return [{ key: 'nonfoil', name: 'Nonfoil' }];
		}
		return getAvailableTreatments(selectedCard.finishes, selectedCard.frame_effects || []);
	});

	async function handleSearch() {
		if (searchQuery.length < 3) {
			searchResults = [];
			return;
		}

		isSearching = true;

		try {
			const response = await fetch(`/api/search?q=${encodeURIComponent(searchQuery)}&page=1`);

			if (response.ok) {
				const data = await response.json();
				searchResults = data.data?.slice(0, 10) || []; // Limit to 10 results
			} else {
				searchResults = [];
			}
		} catch (error) {
			searchResults = [];
		} finally {
			isSearching = false;
		}
	}

	function handleSearchInput() {
		// Clear any existing timeout
		if (searchTimeout) {
			clearTimeout(searchTimeout);
		}

		// Set new timeout for search
		searchTimeout = setTimeout(() => {
			handleSearch();
		}, 300); // Wait 300ms after user stops typing
	}

	async function selectCard(card: ScryfallCard) {
		searchResults = [];
		searchQuery = card.name;
		matchedRule = null;
		matchedLocation = null;
		evaluationError = '';

		// Fetch full card data via SvelteKit API (proxies to backend -> Scryfall) for accurate rule evaluation
		try {
			const response = await fetch(`/api/cards/${card.id}`);
			if (response.ok) {
				const fullCard = await response.json();
				selectedCard = fullCard;

				// Auto-select treatment based on available finishes
				if (fullCard.finishes && fullCard.finishes.length > 0) {
					selectedTreatment = fullCard.finishes[0];
				} else {
					selectedTreatment = 'nonfoil';
				}

				// Auto-evaluate when card is selected
				evaluateRules();
			} else {
				selectedCard = card;
				selectedTreatment = 'nonfoil';
				evaluationError = 'Could not fetch full card data. Evaluation may be incomplete.';
			}
		} catch (error) {
			selectedCard = card;
			selectedTreatment = 'nonfoil';
			evaluationError = 'Could not fetch full card data. Evaluation may be incomplete.';
		}
	}

	async function evaluateRules() {
		if (!selectedCard) return;

		isEvaluating = true;
		evaluationError = '';
		matchedRule = null;
		matchedLocation = null;

		try {
			const response = await fetch(`/api/sorting-rules/evaluate`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					card_data: selectedCard,
					treatment: selectedTreatment
				})
			});

			if (response.ok) {
				const result = await response.json();

				if (result.storage_location) {
					matchedLocation = result.storage_location;

					// Find the matching rule by storage location ID
					const foundRule = rules.find((r) => r.storage_location_id === result.storage_location.id);

					if (foundRule) {
						matchedRule = foundRule;

						// Notify parent component about the match
						if (onRuleMatch) {
							onRuleMatch(foundRule.id);
						}
					}
				}
			} else {
				const error = await response.json();
				evaluationError = error.error || 'Failed to evaluate rules';
			}
		} catch (error) {
			evaluationError = 'Failed to evaluate rules';
		} finally {
			isEvaluating = false;
		}
	}

	function clearSelection() {
		selectedCard = null;
		selectedTreatment = 'nonfoil';
		matchedRule = null;
		matchedLocation = null;
		searchQuery = '';
		searchResults = [];
		evaluationError = '';
	}
</script>

<div class="card bg-base-100 shadow-sm border border-base-300">
	<div class="card-body p-4 space-y-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-2 text-sm font-semibold">
				<Zap class="w-4 h-4 text-warning" />
				<span>Test Your Rules</span>
			</div>
			{#if selectedCard}
				<button type="button" onclick={clearSelection} class="btn btn-ghost btn-xs"> Clear </button>
			{/if}
		</div>

		<!-- Card Search -->
		{#if !selectedCard}
			<div class="form-control">
				<label for="rule-tester-search" class="label">
					<span class="label-text text-xs">Search for a card</span>
				</label>
				<div class="relative">
					<input
						id="rule-tester-search"
						type="text"
						placeholder="e.g., Lightning Bolt"
						bind:value={searchQuery}
						oninput={handleSearchInput}
						class="input input-bordered input-sm w-full pr-10" />
					<Search class="absolute right-3 top-2.5 w-4 h-4 opacity-50" />
					{#if isSearching}
						<span class="absolute right-10 top-2.5 loading loading-spinner loading-sm"></span>
					{/if}
				</div>
			</div>

			<!-- Search Results -->
			{#if searchResults.length > 0}
				<div class="max-h-60 overflow-y-auto space-y-1">
					{#each searchResults as card (card.id)}
						<button
							type="button"
							onclick={() => selectCard(card)}
							class="btn btn-ghost btn-sm w-full justify-start text-left h-auto py-2 normal-case">
							<div class="flex items-center gap-3 w-full">
								{#if card.image_uris?.small}
									<img src={card.image_uris.small} alt={card.name} class="w-10 h-auto rounded" />
								{/if}
								<div class="flex-1 min-w-0">
									<div class="font-semibold text-xs truncate">{card.name}</div>
									<div class="text-xs opacity-70">
										{card.set_name} · {card.rarity}
										{#if card.prices?.usd}
											· ${card.prices.usd}
										{/if}
									</div>
								</div>
							</div>
						</button>
					{/each}
				</div>
			{:else if searchQuery.length >= 3 && !isSearching}
				<div class="text-center py-4 text-sm opacity-70">No cards found</div>
			{/if}
		{/if}

		<!-- Selected Card Display -->
		{#if selectedCard}
			<div class="card bg-base-200 shadow-sm">
				<div class="card-body p-3 space-y-2">
					<div class="flex items-start gap-3">
						{#if selectedCard.image_uris?.small}
							<img
								src={selectedCard.image_uris.small}
								alt={selectedCard.name}
								class="w-20 h-auto rounded" />
						{/if}
						<div class="flex-1 min-w-0">
							<h3 class="font-semibold text-sm">{selectedCard.name}</h3>
							<div class="text-xs opacity-70 space-y-0.5 mt-1">
								<div><strong>Set:</strong> {selectedCard.set_name}</div>
								<div><strong>Rarity:</strong> {selectedCard.rarity}</div>
								{#if selectedCard.prices?.usd}
									<div><strong>Price:</strong> ${selectedCard.prices.usd}</div>
								{/if}
								{#if selectedCard.colors && selectedCard.colors.length > 0}
									<div><strong>Colors:</strong> {selectedCard.colors.join(', ')}</div>
								{:else}
									<div><strong>Colors:</strong> Colorless</div>
								{/if}
								{#if selectedCard.cmc !== undefined}
									<div><strong>CMC:</strong> {selectedCard.cmc}</div>
								{/if}
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Treatment Selector -->
			{#if availableTreatments.length > 0}
				<div class="form-control">
					<label for="treatment-select" class="label">
						<span class="label-text text-xs">Select Treatment</span>
					</label>
					<select
						id="treatment-select"
						bind:value={selectedTreatment}
						onchange={evaluateRules}
						class="select select-bordered select-sm w-full">
						{#each availableTreatments as treatment (treatment.key)}
							<option value={treatment.key}>{treatment.name}</option>
						{/each}
					</select>
				</div>
			{/if}

			<!-- Evaluation Result -->
			{#if isEvaluating}
				<div class="alert alert-info py-2">
					<span class="loading loading-spinner loading-sm"></span>
					<span class="text-xs">Evaluating rules...</span>
				</div>
			{:else if evaluationError}
				<div class="alert alert-error py-2">
					<AlertCircle class="w-4 h-4" />
					<span class="text-xs">{evaluationError}</span>
				</div>
			{:else if matchedRule && matchedLocation}
				{@const treatmentName = getCardTreatmentName(
					selectedCard?.finishes || [],
					selectedCard?.frame_effects || [],
					selectedTreatment
				)}
				<div class="alert alert-success py-2">
					<CheckCircle2 class="w-4 h-4" />
					<div class="flex-1 text-xs">
						<div class="font-semibold">Matched Rule: {matchedRule.name}</div>
						<div class="opacity-90 mt-0.5">
							Storage Location: <strong>{matchedLocation.name}</strong>
						</div>
						<div class="opacity-90 mt-0.5">
							Treatment: <strong>{treatmentName}</strong>
						</div>
						<div class="opacity-70 mt-1">
							Expression: <code class="text-xs">{matchedRule.expression}</code>
						</div>
					</div>
				</div>
			{:else if selectedCard}
				{@const treatmentName = getCardTreatmentName(
					selectedCard?.finishes || [],
					selectedCard?.frame_effects || [],
					selectedTreatment
				)}
				<div class="alert alert-warning py-2">
					<XCircle class="w-4 h-4" />
					<div class="text-xs">
						<div class="font-semibold">No rules matched this card</div>
						<div class="opacity-80 mt-0.5">
							This <strong>{treatmentName}</strong> card would not be automatically assigned to a storage
							location.
						</div>
					</div>
				</div>
			{/if}

			<!-- Re-evaluate Button -->
			{#if selectedCard && !isEvaluating}
				<button type="button" onclick={evaluateRules} class="btn btn-sm btn-primary w-full">
					<Zap class="w-4 h-4" />
					Re-evaluate
				</button>
			{/if}
		{/if}

		<!-- Help Text -->
		{#if !selectedCard && searchQuery.length < 3}
			<div class="text-center py-6 text-xs opacity-70">
				<div class="mb-2">
					<Search class="w-8 h-8 mx-auto opacity-50" />
				</div>
				<p>Search for a card to test your sorting rules</p>
				<p class="mt-1">Enter at least 3 characters to search</p>
			</div>
		{/if}
	</div>
</div>
