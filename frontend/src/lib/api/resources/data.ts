import { apiClient } from '../client';
import type { ExportData, ImportResponse } from '$lib/types/api';

/**
 * Data import/export API methods
 */
export const dataApi = {
	/**
	 * Import data from an export file
	 */
	import: (data: ExportData) => apiClient.post<ImportResponse>('/data/import', data)
};
