import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { ScheduledTaskInfo } from '#lib';
import ScheduledTaskCard from './ScheduledTaskCard.svelte';

function task(overrides: Partial<ScheduledTaskInfo> = {}): ScheduledTaskInfo {
	return {
		name: 'data_backup',
		display_name: 'Data Backup',
		description: 'Exports your collection to a backup file.',
		enabled: true,
		time: '03:30',
		interval_days: 1,
		schedule: '03:30 daily',
		next_run: new Date(Date.now() + 2 * 3600_000).toISOString(),
		...overrides
	};
}

describe('ScheduledTaskCard', () => {
	it('renders the registry display name, description and schedule', async () => {
		await render(ScheduledTaskCard, { task: task() });

		await expect.element(page.getByRole('heading', { name: 'Data Backup' })).toBeInTheDocument();
		await expect
			.element(page.getByText('Exports your collection to a backup file.'))
			.toBeInTheDocument();
		await expect.element(page.getByText('Schedule: 03:30 daily')).toBeInTheDocument();
		await expect.element(page.getByText('Never run')).toBeInTheDocument();
	});

	it('shows the task as disabled without a next run', async () => {
		await render(ScheduledTaskCard, { task: task({ enabled: false, next_run: undefined }) });

		await expect.element(page.getByText('Task disabled')).toBeInTheDocument();
	});

	it('falls back to the settings status for tasks without job records', async () => {
		await render(ScheduledTaskCard, {
			task: task({ last_run: new Date().toISOString(), last_status: 'success' })
		});

		await expect.element(page.getByText('Succeeded')).toBeInTheDocument();
	});

	it("prefers the last job's status and links to it", async () => {
		await render(ScheduledTaskCard, {
			task: task({
				last_run: new Date().toISOString(),
				last_status: 'success',
				last_job_id: 42,
				last_job_status: 'failed'
			})
		});

		await expect.element(page.getByText('Failed')).toBeInTheDocument();
		await expect
			.element(page.getByRole('link', { name: 'View last job →' }))
			.toHaveAttribute('href', expect.stringContaining('/jobs#job-42'));
	});
});
