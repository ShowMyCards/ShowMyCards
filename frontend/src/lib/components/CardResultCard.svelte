<script lang="ts">
	import { SvelteMap } from 'svelte/reactivity';
	import type { EnhancedCardResult, Inventory, StorageLocation, CardActions } from '$lib';
	import { getCardTreatmentName, getActionError, notifications, keyboard, selection, audio } from '$lib';
	import PriceLozenge from './PriceLozenge.svelte';
	import StorageLocationDropdown from './StorageLocationDropdown.svelte';
	import PrintingConflictModal from './PrintingConflictModal.svelte';
	import TreatmentBadge from './TreatmentBadge.svelte';
	import { deserialize } from '$app/forms';
	import type { ByOracleResponse, ExistingPrintingInfo } from '$lib/types/api';

	let {
		card,
		onremove,
		storageLocations = [],
		selectable = false
	}: {
		card: EnhancedCardResult;
		onremove?: (cardId: string) => void;
		storageLocations?: StorageLocation[];
		selectable?: boolean;
	} = $props();

	// Selected storage location for manual override (default to auto)
	let selectedStorageLocation = $state<number | 'auto'>('auto');

	// Track if this card is the keyboard hover target
	const isKeyboardTarget = $derived(keyboard.hoveredId === card.id);

	// Local reactive state for inventory (initialized from prop, then managed locally)
	// svelte-ignore state_referenced_locally (we're doing this on purpose)
	let inventory = $state<Inventory[]>([...card.inventory.this_printing]);
	// svelte-ignore state_referenced_locally (we're doing this on purpose)
	let totalQuantity = $state(card.inventory.total_quantity);

	let adding = $state(false);

	// Printing conflict modal state
	let showConflictModal = $state(false);
	let conflictTreatment = $state('');
	let existingPrintings = $state<ExistingPrintingInfo[]>([]);
	let existingLocations = $state<StorageLocation[]>([]);

	// Group inventory by treatment for this printing
	const thisPrintingByTreatment = $derived.by(() => {
		const map = new SvelteMap<string, number>();
		for (const inv of inventory) {
			const treatment = inv.treatment || 'nonfoil';
			map.set(treatment, (map.get(treatment) || 0) + inv.quantity);
		}
		return map;
	});

	// Get available treatments for this card
	const availableTreatments = $derived(card.finishes.length > 0 ? card.finishes : ['nonfoil']);

	// Check if there are existing printings in different locations
	async function checkForExistingPrintings(treatment: string): Promise<boolean> {
		try {
			const response = await fetch(`/api/inventory/by-oracle/${card.oracle_id}`);
			if (!response.ok) return false;

			const data: ByOracleResponse = await response.json();

			// Filter to only locations that are different from selected (if not auto)
			// and only include items in assigned locations
			const assignedLocations = data.locations.filter((loc) => loc.id);

			if (assignedLocations.length === 0) return false;

			// If user has selected a specific location, check if any existing ones differ
			if (selectedStorageLocation !== 'auto') {
				const hasConflict = assignedLocations.some((loc) => loc.id !== selectedStorageLocation);
				if (!hasConflict) return false;
			}

			// Show the modal with existing printings info
			existingPrintings = data.printings;
			existingLocations = assignedLocations;
			conflictTreatment = treatment;
			showConflictModal = true;
			return true;
		} catch {
			// If check fails, proceed without showing modal
			return false;
		}
	}

	// Handle user's choice from conflict modal
	function handleConflictChoice(locationId: number | 'auto') {
		showConflictModal = false;
		// Set the selected location and proceed with add
		if (locationId !== 'auto') {
			selectedStorageLocation = locationId;
		}
		doAddToInventory(conflictTreatment);
	}

	async function handleIncrement(treatment: string) {
		if (adding) return;

		// Check for existing printings in different locations
		const hasConflict = await checkForExistingPrintings(treatment);
		if (hasConflict) {
			// Modal is shown, wait for user choice
			return;
		}

		// No conflict, proceed with add
		await doAddToInventory(treatment);
	}

	async function doAddToInventory(treatment: string) {
		if (adding) return;

		adding = true;

		// Optimistically update the UI
		const optimisticInv: Inventory = {
			id: Date.now(), // Temporary ID
			scryfall_id: card.id,
			oracle_id: card.oracle_id,
			treatment: treatment,
			quantity: 1,
			storage_location_id: undefined,
			storage_location: undefined,
			created_at: new Date().toISOString(),
			updated_at: new Date().toISOString()
		};
		inventory = [...inventory, optimisticInv];
		totalQuantity += 1;

		try {
			// Create inventory item via form action
			// If storage location is set to 'auto', backend will auto-assign via rules
			const formData = new FormData();
			formData.append('scryfall_id', card.id);
			formData.append('oracle_id', card.oracle_id);
			formData.append('treatment', treatment);
			formData.append('quantity', '1');
			formData.append('storage_location_id', selectedStorageLocation.toString());

			const response = await fetch('?/addInventory', {
				method: 'POST',
				body: formData
			});

			// Deserialize the form action response properly
			const result = deserialize(await response.text());

			// Handle different result types
			if (result.type === 'error' || result.type === 'redirect') {
				throw new Error('Unexpected response type');
			}

			if (result.type === 'failure') {
				throw new Error(getActionError(result.data, 'Failed to add card to inventory'));
			}

			// Extract inventory item from successful response
			const actionData = result.data as { success: boolean; action: string; data: Inventory };
			const newInv = actionData?.data;

			if (!newInv) {
				throw new Error('No inventory data returned from server');
			}

			// Replace optimistic update with real data
			inventory = [...inventory.filter((i) => i.id !== optimisticInv.id), newInv];

			// Build success message with storage location if assigned
			const treatmentName = getCardTreatmentName(
				card.finishes,
				card.frame_effects || [],
				treatment,
				card.promo_types || []
			);
			let message = `Added ${card.name} (${treatmentName}) to inventory`;
			if (newInv.storage_location) {
				message += ` → ${newInv.storage_location.name}`;
				// Play sound when a rule matched and assigned a storage location
				audio.play('match');
			}

			// Show success notification
			notifications.success(message);
		} catch (e) {
			// Revert optimistic update on error
			inventory = inventory.filter((i) => i.id !== optimisticInv.id);
			totalQuantity -= 1;

			const errorMessage = e instanceof Error ? e.message : 'Failed to add card';
			notifications.error(errorMessage);
		} finally {
			adding = false;
		}
	}

	async function handleDecrement(treatment: string) {
		// Find the inventory item for this treatment
		const inv = inventory.find((i) => i.treatment === treatment);
		if (!inv) return;

		try {
			// Delete inventory item via form action
			const formData = new FormData();
			formData.append('inventory_id', inv.id.toString());

			const response = await fetch('?/deleteInventory', {
				method: 'POST',
				body: formData
			});

			// Deserialize the form action response properly
			const result = deserialize(await response.text());

			// Handle different result types
			if (result.type === 'error' || result.type === 'redirect') {
				throw new Error('Unexpected response type');
			}

			if (result.type === 'failure') {
				throw new Error(getActionError(result.data, 'Failed to remove card from inventory'));
			}

			// Update local state
			inventory = inventory.filter((i) => i.id !== inv.id);
			totalQuantity -= inv.quantity;

			// Show success notification
			const treatmentName = getCardTreatmentName(
				card.finishes,
				card.frame_effects || [],
				treatment,
				card.promo_types || []
			);
			notifications.success(`Removed ${card.name} (${treatmentName}) from inventory`);

			// If no inventory left, signal parent to remove this card from view
			if (totalQuantity === 0 && onremove) {
				onremove(card.id);
			}
		} catch (e) {
			const errorMessage = e instanceof Error ? e.message : 'Failed to remove card';
			notifications.error(errorMessage);
		}
	}

	// Keyboard shortcut actions for this card (Alt+N to add, Alt+Shift+N to remove)
	const cardActions: CardActions = {
		increment: (treatmentIndex: number) => {
			const treatment = availableTreatments[treatmentIndex];
			if (treatment) handleIncrement(treatment);
		},
		decrement: (treatmentIndex: number) => {
			const treatment = availableTreatments[treatmentIndex];
			if (treatment) handleDecrement(treatment);
		},
		get treatmentCount() {
			return availableTreatments.length;
		}
	};

	function handleMouseEnter() {
		keyboard.setHoverTarget(card.id, cardActions);
	}

	function handleMouseLeave() {
		keyboard.clearHoverTarget(card.id);
	}

	// Selection handling - select all inventory IDs for this card
	const inventoryIds = $derived(inventory.map((i) => i.id));
	const isSelected = $derived(inventoryIds.length > 0 && inventoryIds.every((id) => selection.isSelected(id)));
	const isPartiallySelected = $derived(
		!isSelected && inventoryIds.some((id) => selection.isSelected(id))
	);

	function handleSelectionChange() {
		if (isSelected) {
			selection.deselectMany(inventoryIds);
		} else {
			selection.selectMany(inventoryIds);
		}
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="card bg-base-200 p-2 shadow-lg hover:shadow-xl transition-all relative"
	class:ring-2={isKeyboardTarget || isSelected}
	class:ring-primary={isKeyboardTarget && !isSelected}
	class:ring-accent={isSelected}
	onmouseenter={handleMouseEnter}
	onmouseleave={handleMouseLeave}>
	<!-- Selection checkbox -->
	{#if selectable && inventoryIds.length > 0}
		<div class="absolute top-3 left-3 z-10">
			<input
				type="checkbox"
				class="checkbox checkbox-accent checkbox-sm"
				checked={isSelected}
				indeterminate={isPartiallySelected}
				onchange={handleSelectionChange} />
		</div>
	{/if}
	<!-- Clickable image to card details -->
	<a href="/cards/{card.id}" class="block">
		{#if card.image_uri}
			<figure class="cursor-pointer hover:opacity-90 transition-opacity">
				<img src={card.image_uri} alt={card.name} loading="lazy" />
			</figure>
		{:else}
			<div class="w-full aspect-5/7 bg-base-300 flex items-center justify-center cursor-pointer hover:opacity-90">
				<p class="opacity-50">No image</p>
			</div>
		{/if}
	</a>

	<div class="card-body p-4">
		<div class="flex items-center justify-between mb-2">
			<div>
				<!-- Clickable name to card details -->
				<a href="/cards/{card.id}" class="card-title text-lg hover:text-primary transition-colors">
					{card.name}
				</a>
				<div class="text-sm opacity-70">{card.set_name}</div>
			</div>
			{#if totalQuantity > 0}
				<div class="badge badge-primary badge-lg font-semibold">
					{totalQuantity}
				</div>
			{/if}
		</div>

		<!-- Storage location override dropdown -->
		{#if storageLocations.length > 0}
			<div class="mb-2">
				<StorageLocationDropdown
					locations={storageLocations}
					selected={selectedStorageLocation}
					onchange={(v) => (selectedStorageLocation = v)}
					compact />
			</div>
		{/if}

		<div class="flex">
			<!-- For each treatment: -->
			{#each availableTreatments as treatment, index (treatment)}
				{@const quantity = thisPrintingByTreatment.get(treatment) || 0}
				<div class="flex-1 p-1 m-1 flex flex-col items-center rounded">
					<!-- Treatment badge with keyboard hint -->
					<div class="flex items-center gap-1">
						{#if isKeyboardTarget}
							<kbd class="kbd kbd-xs opacity-70">{index + 1}</kbd>
						{/if}
						<TreatmentBadge
							{treatment}
							finishes={card.finishes}
							frameEffects={card.frame_effects ?? []}
							promoTypes={card.promo_types ?? []}
							size="sm" />
					</div>
					<!-- Price -->
					{#if card.prices && (card.prices.usd || card.prices.usd_foil || card.prices.usd_etched)}
						<div class="py-1">
							<PriceLozenge {treatment} prices={card.prices} />
						</div>
					{/if}
					<!-- Quantity (prominent) -->
					<div class="text-2xl font-bold my-1">{quantity}</div>
					<!-- +/- Buttons -->
					<div class="flex items-center gap-1">
						<button
							onclick={() => handleDecrement(treatment)}
							disabled={quantity === 0}
							class="btn btn-sm btn-square bg-base-100">
							−
						</button>
						<button
							onclick={() => handleIncrement(treatment)}
							disabled={adding}
							class="btn btn-sm btn-square bg-base-100">
							+
						</button>
					</div>
				</div>
			{/each}
		</div>
	</div>
</div>

<PrintingConflictModal
	open={showConflictModal}
	cardName={card.name}
	treatment={conflictTreatment}
	{existingPrintings}
	{existingLocations}
	onClose={() => (showConflictModal = false)}
	onChoose={handleConflictChoice}
/>
