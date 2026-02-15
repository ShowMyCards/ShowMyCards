/**
 * Shared server-side form action handlers for inventory add/delete.
 * Used by search, import, inventory/[id], inventory/unassigned, and inventory/recent pages.
 */
import { BACKEND_URL } from '$lib';
import { fail } from '@sveltejs/kit';

/**
 * Handle adding an inventory item. Reads scryfall_id, oracle_id, treatment, quantity,
 * and optionally storage_location_id from the form data.
 */
export async function handleAddInventory(request: Request, fetch: typeof globalThis.fetch) {
	const formData = await request.formData();
	const scryfallId = formData.get('scryfall_id');
	const oracleId = formData.get('oracle_id');
	const treatment = formData.get('treatment');
	const quantity = formData.get('quantity');
	const storageLocationId = formData.get('storage_location_id');

	if (!scryfallId || !oracleId || !treatment) {
		return fail(400, { error: 'Missing required fields' });
	}

	try {
		const body: Record<string, unknown> = {
			scryfall_id: scryfallId,
			oracle_id: oracleId,
			treatment: treatment,
			quantity: quantity ? parseInt(quantity.toString()) : 1
		};

		// Only include storage_location_id if explicitly set (not 'auto')
		if (storageLocationId && storageLocationId !== 'auto') {
			body.storage_location_id = parseInt(storageLocationId.toString());
		}

		const response = await fetch(`${BACKEND_URL}/inventory`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(body)
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			return fail(response.status, {
				error: errorData.error || 'Failed to add to inventory'
			});
		}

		const inventoryItem = await response.json();
		return { success: true, action: 'add' as const, data: inventoryItem };
	} catch {
		return fail(500, { error: 'Failed to add to inventory' });
	}
}

/**
 * Handle deleting an inventory item. Reads inventory_id from the form data.
 */
export async function handleDeleteInventory(request: Request, fetch: typeof globalThis.fetch) {
	const formData = await request.formData();
	const inventoryId = formData.get('inventory_id');

	if (!inventoryId) {
		return fail(400, { error: 'Inventory ID is required' });
	}

	try {
		const response = await fetch(`${BACKEND_URL}/inventory/${inventoryId}`, {
			method: 'DELETE'
		});

		if (!response.ok) {
			return fail(response.status, { error: 'Failed to delete inventory item' });
		}

		return { success: true, action: 'delete' as const };
	} catch {
		return fail(500, { error: 'Failed to delete inventory item' });
	}
}
