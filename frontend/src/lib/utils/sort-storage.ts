/**
 * Shape accepted by {@link compareStorageLocations}.
 * A `null` id represents the synthetic "Unassigned" bucket, which always sorts last.
 */
export interface StorageLocationSortable {
	id: number | null;
	sortOrder?: number | null;
	treatment?: string;
}

/**
 * Compares two storage-location entries in canonical display order:
 * 1. Assigned locations before unassigned (`id === null` last)
 * 2. `sortOrder` ascending (missing values sort after known orders, before unassigned)
 * 3. `id` ascending (matches backend `sort_order ASC, id ASC`)
 * 4. `treatment` lexicographically when both entries share a location
 */
export function compareStorageLocations(
	a: StorageLocationSortable,
	b: StorageLocationSortable
): number {
	if (a.id === null && b.id !== null) return 1;
	if (a.id !== null && b.id === null) return -1;

	const orderA = a.sortOrder ?? Number.MAX_SAFE_INTEGER;
	const orderB = b.sortOrder ?? Number.MAX_SAFE_INTEGER;
	if (orderA !== orderB) return orderA - orderB;

	const idDiff = (a.id ?? 0) - (b.id ?? 0);
	if (idDiff !== 0) return idDiff;

	return (a.treatment ?? '').localeCompare(b.treatment ?? '');
}
