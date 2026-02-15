<script lang="ts">
	import { selection, getCardTreatmentName, scryfallImageUrl } from '$lib';
	import { resolve } from '$app/paths';
	import CardPreview from '../../CardPreview.svelte';
	import SetIcon from '../../SetIcon.svelte';
	import type { DisplayableCard, TableColumn } from '../types';
	import { isFoilTreatment } from '../adapters';

	interface Props {
		items: DisplayableCard[];
		columns?: TableColumn[];
		selectable?: boolean;
		onItemRemove?: (itemId: string | number) => void;
	}

	let { items, columns, selectable = false }: Props = $props();

	// Default columns if none provided
	const defaultColumns: TableColumn[] = [
		{ key: 'name', header: 'Card Name' },
		{ key: 'set', header: 'Set' },
		{ key: 'number', header: '#', width: 'w-16' },
		{ key: 'treatment', header: 'Treatment' },
		{ key: 'quantity', header: 'Qty', width: 'w-16' }
	];

	const effectiveColumns = $derived(columns ?? defaultColumns);

	/**
	 * Get the image URL for a card, constructing from Scryfall ID if needed
	 */
	function getImageUrl(item: DisplayableCard): string | undefined {
		if (item.imageUri) return item.imageUri;
		return scryfallImageUrl(item.scryfallId);
	}

	/**
	 * Get the treatment display name
	 */
	function getTreatmentName(item: DisplayableCard): string {
		return getCardTreatmentName(item.finishes, item.frameEffects ?? [], item.treatment);
	}

	/**
	 * Check if a treatment is a foil variant
	 */
	function isFoil(item: DisplayableCard): boolean {
		return isFoilTreatment(item.treatment);
	}

	/**
	 * Handle row selection toggle
	 */
	function handleSelectionToggle(item: DisplayableCard) {
		if (typeof item.id === 'number') {
			selection.toggle(item.id);
		}
	}

	/**
	 * Check if an item is selected
	 */
	function isSelected(item: DisplayableCard): boolean {
		if (typeof item.id === 'number') {
			return selection.isSelected(item.id);
		}
		return false;
	}

	/**
	 * Get cell value for a column
	 */
	function getCellValue(item: DisplayableCard, column: TableColumn): string | number | undefined {
		if (column.accessor) {
			return column.accessor(item);
		}

		// Default accessors for common columns
		switch (column.key) {
			case 'name':
				return item.name;
			case 'set':
				return item.setName;
			case 'number':
				return item.collectorNumber;
			case 'treatment':
				return getTreatmentName(item);
			case 'quantity':
				return item.quantity;
			default:
				return undefined;
		}
	}
</script>

<div class="overflow-x-auto">
	<table class="table table-zebra">
		<thead>
			<tr>
				{#if selectable}
					<th class="w-10">
						<span class="sr-only">Select</span>
					</th>
				{/if}
				{#each effectiveColumns as column (column.key)}
					<th class={column.width ?? ''}>{column.header}</th>
				{/each}
			</tr>
		</thead>
		<tbody>
			{#each items as item (item.id)}
				{@const imageUrl = getImageUrl(item)}
				{@const treatmentName = getTreatmentName(item)}
				{@const itemIsFoil = isFoil(item)}

				<tr class="hover:bg-base-300" class:bg-accent={isSelected(item)} class:bg-opacity-20={isSelected(item)}>
					{#if selectable}
						<td>
							<input
								type="checkbox"
								class="checkbox checkbox-sm checkbox-accent"
								checked={isSelected(item)}
								onchange={() => handleSelectionToggle(item)} />
						</td>
					{/if}

					{#each effectiveColumns as column (column.key)}
						<td class={column.width ?? ''}>
							{#if column.render}
								{@render column.render(item)}
							{:else if column.key === 'name'}
								<CardPreview src={imageUrl} alt={item.name} isFoil={itemIsFoil}>
									<a href={resolve(`/cards/${item.scryfallId}`)} class="font-semibold hover:text-primary transition-colors">
										{item.name}
									</a>
								</CardPreview>
							{:else if column.key === 'set'}
								<div class="flex items-center gap-2">
									{#if item.setCode}
										<SetIcon
											setCode={item.setCode}
											setName={item.setName ?? ''}
											rarity={item.rarity ?? 'common'}
											isFoil={itemIsFoil} />
									{/if}
									<span class="text-sm">{item.setName ?? ''}</span>
								</div>
							{:else if column.key === 'treatment'}
								<span
									class="text-sm font-medium px-2 py-0.5 rounded"
									class:bg-gradient-to-r={itemIsFoil}
									class:from-yellow-200={itemIsFoil}
									class:to-amber-300={itemIsFoil}
									class:text-amber-900={itemIsFoil}>
									{treatmentName}
								</span>
							{:else if column.key === 'quantity'}
								<span class="badge badge-primary">{item.quantity}</span>
							{:else}
								{getCellValue(item, column) ?? ''}
							{/if}
						</td>
					{/each}
				</tr>
			{/each}
		</tbody>
	</table>
</div>
