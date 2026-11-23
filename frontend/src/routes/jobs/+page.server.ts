import { BACKEND_URL, type JobRecord, type ScheduledTaskInfo } from '#lib';
import type { PageServerLoad } from './$types';

interface PaginatedResponse {
	data: JobRecord[];
	page: number;
	page_size: number;
	total_items: number;
	total_pages: number;
}

export const load: PageServerLoad = async ({ fetch, url }) => {
	try {
		const page = url.searchParams.get('page') || '1';

		// Fetch jobs and scheduled tasks in parallel
		const [jobsResponse, tasksResponse] = await Promise.all([
			fetch(`${BACKEND_URL}/api/jobs?page=${page}&page_size=20`),
			fetch(`${BACKEND_URL}/api/scheduler/tasks`)
		]);

		if (!jobsResponse.ok) {
			return {
				jobs: [] as JobRecord[],
				scheduledTasks: [] as ScheduledTaskInfo[],
				page: 1,
				totalPages: 0,
				error: 'Failed to load jobs'
			};
		}

		const jobsResult: PaginatedResponse = await jobsResponse.json();
		const scheduledTasks: ScheduledTaskInfo[] = tasksResponse.ok ? await tasksResponse.json() : [];

		return {
			jobs: jobsResult.data || [],
			scheduledTasks: scheduledTasks,
			page: jobsResult.page || 1,
			totalPages: jobsResult.total_pages || 0,
			error: null
		};
	} catch {
		return {
			jobs: [] as JobRecord[],
			scheduledTasks: [] as ScheduledTaskInfo[],
			page: 1,
			totalPages: 0,
			error: 'Failed to load jobs'
		};
	}
};
