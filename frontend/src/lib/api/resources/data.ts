import { apiClient } from '../client';
import { BACKEND_URL } from '$lib/config';
import type { ExportData, ImportResponse } from '$lib/types/api';

/**
 * Data import/export API methods
 */
export const dataApi = {
	/**
	 * Get the full URL for the export endpoint (for direct browser download)
	 */
	exportUrl: () => `${BACKEND_URL}/api/data/export`,

	/**
	 * Import data from an export file
	 */
	import: (data: ExportData) => apiClient.post<ImportResponse>('/api/data/import', data)
};
