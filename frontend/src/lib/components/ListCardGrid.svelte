<script lang="ts">
	import { enhance } from '$app/forms';
	import {
		Pagination,
		CardImage,
		TreatmentBadge,
		isFoilTreatment,
		notifications,
		getActionError,
		scryfallImageUrl,
		type EnrichedListItem
	} from '$lib';
	import { Plus, Minus } from '@lucide/svelte';

	interface Props {
		items: EnrichedListItem[];
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}

	let { items, currentPage, totalPages, onPageChange }: Props = $props();
</script>

<div class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6 gap-4">
	{#each items as item (item.id)}
		{@const isFoil = isFoilTreatment(item.treatment)}
		{@const progress =
			item.desired_quantity > 0 ? (item.collected_quantity / item.desired_quantity) * 100 : 0}
		{@const isComplete = item.collected_quantity >= item.desired_quantity}
		{@const isUncollected = item.collected_quantity === 0}
		<div class="card bg-base-200 shadow-sm overflow-hidden group relative">
			<!-- Card image with foil effect and greyed out for uncollected -->
			<figure class="relative">
				<CardImage
					src={scryfallImageUrl(item.scryfall_id)}
					alt={item.name || 'Card'}
					{isFoil}
					class="w-full aspect-488/680 {isUncollected ? 'grayscale opacity-60' : ''}" />
				<!-- Progress overlay -->
				<div
					class="absolute bottom-0 left-0 right-0 bg-base-300/90 p-2 flex items-center justify-between">
					<span class="text-xs font-medium">
						{item.collected_quantity}/{item.desired_quantity}
					</span>
					{#if isComplete}
						<span class="badge badge-xs badge-success">Complete</span>
					{:else}
						<progress
							class="progress w-16 h-2 {isUncollected ? 'progress-error' : 'progress-warning'}"
							value={progress}
							max="100"></progress>
					{/if}
				</div>
			</figure>
			<!-- Card info -->
			<div class="p-2">
				<p class="text-xs font-semibold truncate" title={item.name}>{item.name}</p>
				<div class="flex items-center gap-1 mt-1">
					<span class="text-xs opacity-70 truncate flex-1">{item.set_name}</span>
					<TreatmentBadge
						treatment={item.treatment}
						finishes={item.finishes ?? [item.treatment]}
						frameEffects={item.frame_effects ?? []}
						promoTypes={item.promo_types ?? []}
						size="xs" />
				</div>
			</div>
			<!-- Action buttons (visible on hover) -->
			<div
				class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity flex gap-1">
				<form
					method="POST"
					action="?/updateItem"
					use:enhance={() => {
						return async ({ result, update }) => {
							await update();
							if (result.type === 'success') {
								notifications.success('Updated!');
							} else if (result.type === 'failure') {
								const errorMsg = getActionError(result.data, 'Failed to update');
								notifications.error(errorMsg);
							}
						};
					}}>
					<input type="hidden" name="item_id" value={item.id} />
					<div class="join join-vertical">
						<button
							type="submit"
							name="collected_quantity"
							value={Math.min(item.desired_quantity, item.collected_quantity + 1)}
							class="btn btn-xs btn-primary join-item"
							title="Increase collected">
							<Plus class="w-3 h-3" />
						</button>
						<button
							type="submit"
							name="collected_quantity"
							value={Math.max(0, item.collected_quantity - 1)}
							class="btn btn-xs btn-primary join-item"
							title="Decrease collected">
							<Minus class="w-3 h-3" />
						</button>
					</div>
				</form>
			</div>
		</div>
	{/each}
</div>
<!-- Pagination for grid view -->
{#if totalPages > 1}
	<div class="mt-6">
		<Pagination {currentPage} {totalPages} {onPageChange} />
	</div>
{/if}
