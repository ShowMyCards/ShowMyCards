<script lang="ts">
	import { CardResultCard, type EnhancedCardResult, type StorageLocation } from '$lib';
	import type { DisplayableCard } from '../types';

	interface Props {
		items: DisplayableCard[];
		storageLocations?: StorageLocation[];
		selectable?: boolean;
		onItemRemove?: (itemId: string | number) => void;
	}

	let { items, storageLocations = [], selectable = false, onItemRemove }: Props = $props();

	function handleRemove(cardId: string) {
		onItemRemove?.(cardId);
	}

	/**
	 * Extract the original EnhancedCardResult from a DisplayableCard
	 * Only works for inventory and search source types
	 */
	function getEnhancedCard(item: DisplayableCard): EnhancedCardResult | null {
		if (item._sourceType === 'inventory' || item._sourceType === 'search') {
			return item._source as EnhancedCardResult;
		}
		return null;
	}
</script>

<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
	{#each items as item (item.id)}
		{@const card = getEnhancedCard(item)}
		{#if card}
			<CardResultCard {card} onremove={handleRemove} {storageLocations} {selectable} />
		{:else}
			<!-- Fallback for list items or other source types -->
			<div class="card bg-base-200 p-4">
				<p class="font-semibold">{item.name}</p>
				<p class="text-sm opacity-70">{item.setName}</p>
				<p class="text-sm">Quantity: {item.quantity}</p>
			</div>
		{/if}
	{/each}
</div>
