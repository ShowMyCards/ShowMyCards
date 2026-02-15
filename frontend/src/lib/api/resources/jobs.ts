import { apiClient } from '../client';
import type { Job } from '$lib';

/**
 * Query parameters for listing jobs
 */
export interface ListJobsParams {
	status?: 'pending' | 'in_progress' | 'completed' | 'failed' | 'cancelled';
}

/**
 * Jobs API methods
 *
 * Provides type-safe access to job endpoints.
 *
 * @example
 * ```ts
 * import { jobsApi } from '$lib/api';
 *
 * // List all jobs
 * const jobs = await jobsApi.list();
 *
 * // Get completed jobs only
 * const completedJobs = await jobsApi.list({ status: 'completed' });
 *
 * // Get a specific job
 * const job = await jobsApi.get(1);
 * ```
 */
export const jobsApi = {
	/**
	 * List all jobs with optional status filter
	 */
	list: (params?: ListJobsParams) => {
		const searchParams = new URLSearchParams();
		if (params?.status) searchParams.set('status', params.status);

		const query = searchParams.toString();
		return apiClient.get<Job[]>(`/jobs${query ? `?${query}` : ''}`);
	},

	/**
	 * Get a single job by ID
	 */
	get: (id: number) => apiClient.get<Job>(`/jobs/${id}`)
};
