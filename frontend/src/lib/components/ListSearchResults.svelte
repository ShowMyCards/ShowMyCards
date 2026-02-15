<script lang="ts">
	import {
		EmptyState,
		getCardTreatmentName,
		sortCardsBySetAndCollector,
		groupCardsByTreatment,
		type ScryfallCard
	} from '$lib';

	interface Props {
		searchResults: ScryfallCard[];
		searchQuery: string;
		adding: boolean;
		isInList: (scryfallId: string, treatment: string) => boolean;
		onBulkAddTreatment: (treatment: string | null) => void;
	}

	let { searchResults, searchQuery, adding, isInList, onBulkAddTreatment }: Props = $props();

	const sortedSearchResults = $derived(sortCardsBySetAndCollector(searchResults));
	const groupedResults = $derived(groupCardsByTreatment(sortedSearchResults));
	const uniqueTreatments = $derived(groupedResults.treatments);
	const cardsByTreatment = $derived(groupedResults.cardsByTreatment);
</script>

{#if sortedSearchResults.length > 0}
	<div class="divider">Search Results ({sortedSearchResults.length} cards)</div>

	<!-- Search Results Tabs -->
	<div class="tabs tabs-boxed mb-4">
		<!-- All Cards Tab -->
		<input type="radio" name="search_tabs" class="tab" aria-label="All Cards" checked={true} />
		<div class="tab-content bg-base-200 border-base-300 rounded-box p-4">
			<div class="flex items-center justify-between mb-4">
				<h3 class="font-semibold">All Cards ({sortedSearchResults.length})</h3>
				<button
					type="button"
					class="btn btn-sm btn-primary"
					onclick={() => onBulkAddTreatment(null)}
					disabled={adding}
					title="Adds 1 of each treatment (nonfoil, foil, etc.) for every card">
					{adding ? 'Adding...' : 'Add All Treatments'}
				</button>
			</div>

			<div class="overflow-x-auto">
				<table class="table table-zebra table-sm">
					<thead>
						<tr>
							<th>Card Name</th>
							<th>Set</th>
							<th>Number</th>
							<th>Treatments</th>
							<th>Status</th>
						</tr>
					</thead>
					<tbody>
						{#each sortedSearchResults as card (card.id)}
							{@const anyInList = (card.finishes || ['nonfoil']).some((t: string) =>
								isInList(card.id, t)
							)}
							<tr class="hover">
								<td class="font-semibold">{card.name}</td>
								<td>{card.set_name}</td>
								<td>#{card.collector_number}</td>
								<td>
									<div class="flex flex-wrap gap-1">
										{#each card.finishes || ['nonfoil'] as treatment (treatment)}
											{@const treatmentName = getCardTreatmentName(
												card.finishes || ['nonfoil'],
												card.frame_effects || [],
												treatment
											)}
											<span class="badge badge-sm badge-outline capitalize">{treatmentName}</span>
										{/each}
									</div>
								</td>
								<td>
									{#if anyInList}
										<span class="badge badge-sm badge-success">In List</span>
									{/if}
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>

		<!-- Treatment-specific tabs -->
		{#each uniqueTreatments as treatment (treatment)}
			{@const treatmentCards = cardsByTreatment.get(treatment) || []}
			<input type="radio" name="search_tabs" class="tab capitalize" aria-label={treatment} />
			<div class="tab-content bg-base-200 border-base-300 rounded-box p-4">
				<div class="flex items-center justify-between mb-4">
					<h3 class="font-semibold capitalize">
						{treatment} Cards ({treatmentCards.length})
					</h3>
					<button
						type="button"
						class="btn btn-sm btn-primary"
						onclick={() => onBulkAddTreatment(treatment)}
						disabled={adding}>
						{adding ? 'Adding...' : `Add 1 of Each ${treatment}`}
					</button>
				</div>

				<div class="overflow-x-auto">
					<table class="table table-zebra table-sm">
						<thead>
							<tr>
								<th>Card Name</th>
								<th>Set</th>
								<th>Number</th>
								<th>Treatment</th>
								<th>Status</th>
							</tr>
						</thead>
						<tbody>
							{#each treatmentCards as card (card.id)}
								{@const treatmentName = getCardTreatmentName(
									card.finishes || ['nonfoil'],
									card.frame_effects || [],
									treatment
								)}
								<tr class="hover">
									<td class="font-semibold">{card.name}</td>
									<td>{card.set_name}</td>
									<td>#{card.collector_number}</td>
									<td>
										<span class="badge badge-sm badge-outline capitalize">{treatmentName}</span>
									</td>
									<td>
										{#if isInList(card.id, treatment)}
											<span class="badge badge-sm badge-success">In List</span>
										{/if}
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</div>
			</div>
		{/each}
	</div>
{:else if searchQuery}
	<EmptyState message="No cards found" />
{/if}
