import { BACKEND_URL, type JobType, type JobStatus, type ScheduledTaskInfo } from '$lib';
import type { PageServerLoad } from './$types';

// API response has flattened BaseModel fields
interface JobResponse {
	id: number;
	created_at: string;
	updated_at: string;
	type: JobType;
	status: JobStatus;
	started_at?: string;
	completed_at?: string;
	error?: string;
	metadata?: string;
}

interface PaginatedResponse {
	data: JobResponse[];
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
				jobs: [] as JobResponse[],
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
	} catch (error) {
		return {
			jobs: [] as JobResponse[],
			scheduledTasks: [] as ScheduledTaskInfo[],
			page: 1,
			totalPages: 0,
			error: 'Failed to load jobs'
		};
	}
};
