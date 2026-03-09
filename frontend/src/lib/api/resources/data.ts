import type { ExportData, ImportResponse } from '$lib/types/api';

/**
 * Data import/export API methods
 */
export const dataApi = {
	/**
	 * Get the URL for the export endpoint (proxied through SvelteKit)
	 */
	exportUrl: () => '/api/data/export',

	/**
	 * Import data from an export file
	 */
	import: async (data: ExportData): Promise<ImportResponse> => {
		const response = await fetch('/api/data/import', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify(data)
		});

		if (!response.ok) {
			const errorData = await response.json().catch(() => ({}));
			throw new Error(errorData.message || `Import failed: ${response.statusText}`);
		}

		return response.json();
	}
};
