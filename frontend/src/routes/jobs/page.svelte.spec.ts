import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { JobRecord, ScheduledTaskInfo } from '#lib';
import Page from './+page.svelte';
import type { PageData } from './$types';

// Kit's own runtime imports afterNavigate from this module, so it must be stubbed too.
vi.mock('$app/navigation', () => ({
	goto: vi.fn(),
	invalidateAll: vi.fn(),
	afterNavigate: vi.fn()
}));

function task(name: string, display_name: string, job_type: string): ScheduledTaskInfo {
	return {
		name,
		display_name,
		description: '',
		enabled: true,
		enabled_key: `${name}_enabled`,
		time: '03:00',
		time_key: `${name}_time`,
		interval_days: 1,
		schedule: '03:00 daily',
		job_type,
		extras: []
	};
}

function job(id: number, type: string, overrides: Partial<JobRecord> = {}): JobRecord {
	return {
		id,
		created_at: '2026-11-20T03:00:00Z',
		updated_at: '2026-11-20T03:00:00Z',
		type,
		status: 'completed',
		started_at: '2026-11-20T03:00:00Z',
		completed_at: '2026-11-20T03:01:00Z',
		...overrides
	};
}

function makeData(jobs: JobRecord[]): PageData {
	return {
		jobs,
		scheduledTasks: [
			task('bulk_data_update', 'Card Data Update', 'bulk_data_import'),
			task('data_backup', 'Data Backup', 'data_backup')
		],
		page: 1,
		totalPages: 1,
		error: null
	} as PageData;
}

describe('/jobs/+page.svelte', () => {
	it('labels job types from the scheduler registry, humanizing unknown types', async () => {
		await render(Page, {
			data: makeData([
				job(1, 'bulk_data_import', {
					metadata: JSON.stringify({ processed_cards: 50, total_cards: 200 })
				}),
				job(2, 'data_backup'),
				job(3, 'mystery_task')
			])
		});

		const table = page.getByRole('table');
		await expect.element(table.getByText('Card Data Update')).toBeInTheDocument();
		await expect.element(table.getByText('Data Backup')).toBeInTheDocument();
		await expect.element(table.getByText('Mystery Task')).toBeInTheDocument();
		await expect.element(table.getByText('50 / 200 (25%)')).toBeInTheDocument();
	});

	it('opens the detail view for any job', async () => {
		await render(Page, {
			data: makeData([
				job(7, 'data_backup', {
					metadata: JSON.stringify({ file_name: 'backup.json.gz', size_bytes: 1536 })
				})
			])
		});

		await page.getByRole('button', { name: 'Details for job 7' }).click();

		await expect.element(page.getByText('Job #7')).toBeInTheDocument();
		await expect
			.element(page.getByRole('region', { name: 'Output' }).getByText('(1.5 KB)'))
			.toBeInTheDocument();
	});
});
