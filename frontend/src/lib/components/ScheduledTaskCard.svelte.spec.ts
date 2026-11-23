import { page } from 'vitest/browser';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { notifications, type ScheduledTaskInfo } from '#lib';
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
	afterEach(() => vi.restoreAllMocks());

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

	it('starts the task through the proxy and reports success', async () => {
		const fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(new Response(JSON.stringify({ job_id: 7 }), { status: 202 }));
		const success = vi.spyOn(notifications, 'success');
		const onStarted = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onStarted });

		await page.getByRole('button', { name: 'Run now' }).click();

		await vi.waitFor(() => expect(onStarted).toHaveBeenCalled());
		expect(fetchSpy).toHaveBeenCalledWith('/api/scheduler/tasks/data_backup/run', {
			method: 'POST'
		});
		expect(success).toHaveBeenCalledWith('Data Backup started');
	});

	it('shows the backend error when the task is already running', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			new Response(JSON.stringify({ error: 'task is already running' }), { status: 409 })
		);
		const error = vi.spyOn(notifications, 'error');
		const onStarted = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onStarted });

		await page.getByRole('button', { name: 'Run now' }).click();

		await vi.waitFor(() => expect(error).toHaveBeenCalledWith('task is already running'));
		expect(onStarted).toHaveBeenCalled();
	});

	it('disables Run now while the task is running', async () => {
		await render(ScheduledTaskCard, {
			task: task({ last_job_id: 3, last_job_status: 'in_progress' })
		});

		await expect.element(page.getByRole('button', { name: 'Run now' })).toBeDisabled();
	});
});
