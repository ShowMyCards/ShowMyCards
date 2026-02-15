<script lang="ts">
	import { enhance } from '$app/forms';
	import {
		Pagination,
		TreatmentBadge,
		isFoilTreatment,
		notifications,
		getActionError,
		type EnrichedListItem
	} from '$lib';
	import { Plus, Minus, X } from '@lucide/svelte';
	import SetIcon from './SetIcon.svelte';

	interface Props {
		items: EnrichedListItem[];
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}

	let { items, currentPage, totalPages, onPageChange }: Props = $props();
</script>

<div class="overflow-x-auto">
	<table class="table table-zebra">
		<thead>
			<tr>
				<th>Card Name</th>
				<th>Set</th>
				<th>#</th>
				<th>Treatment</th>
				<th>Progress</th>
				<th>Actions</th>
			</tr>
		</thead>
		<tbody>
			{#each items as item (item.id)}
				{@const isFoil = isFoilTreatment(item.treatment)}
				{@const progress =
					item.desired_quantity > 0 ? (item.collected_quantity / item.desired_quantity) * 100 : 0}
				{@const isComplete = item.collected_quantity >= item.desired_quantity}
				{@const isUncollected = item.collected_quantity === 0}
				<tr class="hover:bg-base-300" class:opacity-60={isUncollected}>
					<td class="font-semibold">{item.name || item.scryfall_id}</td>
					<td>
						<div class="flex items-center gap-2">
							<SetIcon
								setName={item.set_name}
								setCode={item.set_code}
								rarity={item.rarity}
								{isFoil} />
							<span class="text-sm">{item.set_name}</span>
						</div>
					</td>
					<td>#{item.collector_number || '?'}</td>
					<td>
						<TreatmentBadge
							treatment={item.treatment}
							finishes={item.finishes ?? [item.treatment]}
							frameEffects={item.frame_effects ?? []}
							promoTypes={item.promo_types ?? []}
							size="sm" />
					</td>
					<td>
						<div class="flex items-center gap-2">
							<span class="text-sm">{item.collected_quantity}/{item.desired_quantity}</span>
							{#if isComplete}
								<span class="badge badge-xs badge-success">Complete</span>
							{:else}
								<progress
									class="progress w-16 h-2 {isUncollected ? 'progress-error' : 'progress-warning'}"
									value={progress}
									max="100"></progress>
							{/if}
						</div>
					</td>
					<td>
						<div class="flex items-center gap-2">
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
								<div class="join">
									<button
										type="submit"
										name="collected_quantity"
										value={Math.max(0, item.collected_quantity - 1)}
										class="btn btn-xs join-item"
										title="Decrease collected">
										<Minus class="w-3 h-3" />
									</button>
									<button
										type="submit"
										name="collected_quantity"
										value={Math.min(item.desired_quantity, item.collected_quantity + 1)}
										class="btn btn-xs join-item"
										title="Increase collected">
										<Plus class="w-3 h-3" />
									</button>
								</div>
							</form>

							<form
								method="POST"
								action="?/deleteItem"
								use:enhance={() => {
									return async ({ result, update }) => {
										await update();
										if (result.type === 'success') {
											notifications.success('Removed from list!');
										} else if (result.type === 'failure') {
											const errorMsg = getActionError(result.data, 'Failed to remove');
											notifications.error(errorMsg);
										}
									};
								}}>
								<input type="hidden" name="item_id" value={item.id} />
								<button
									type="submit"
									class="btn btn-xs btn-ghost text-error"
									title="Remove from list">
									<X class="w-3 h-3" />
								</button>
							</form>
						</div>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</div>
<!-- Pagination for table view -->
{#if totalPages > 1}
	<div class="mt-6">
		<Pagination {currentPage} {totalPages} {onPageChange} />
	</div>
{/if}
