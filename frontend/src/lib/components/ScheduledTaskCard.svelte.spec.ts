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
		enabled_key: 'data_backup_enabled',
		time_key: 'data_backup_time',
		extras: [
			{
				key: 'data_backup_retention_count',
				label: 'Backups to keep',
				description: '0 keeps every backup.',
				kind: 'int',
				value: '14',
				min: 0
			}
		],
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
		const onUpdated = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onUpdated });

		await page.getByRole('button', { name: 'Run now' }).click();

		await vi.waitFor(() => expect(onUpdated).toHaveBeenCalled());
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
		const onUpdated = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onUpdated });

		await page.getByRole('button', { name: 'Run now' }).click();

		await vi.waitFor(() => expect(error).toHaveBeenCalledWith('task is already running'));
		expect(onUpdated).toHaveBeenCalled();
	});

	it('disables Run now while the task is running', async () => {
		await render(ScheduledTaskCard, {
			task: task({ last_job_id: 3, last_job_status: 'in_progress' })
		});

		await expect.element(page.getByRole('button', { name: 'Run now' })).toBeDisabled();
	});

	it('saves the edited schedule and extras through the settings API', async () => {
		const fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(new Response(JSON.stringify({ message: 'ok' }), { status: 200 }));
		const onUpdated = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onUpdated });

		await page.getByRole('button', { name: 'Edit schedule' }).click();
		await page.getByRole('checkbox', { name: 'Enabled' }).click();
		await page.getByLabelText(/^Time of day/).fill('04:15');
		await page.getByLabelText(/^Backups to keep/).fill('7');
		await page.getByRole('button', { name: 'Save' }).click();

		await vi.waitFor(() => expect(onUpdated).toHaveBeenCalled());
		const [url, init] = fetchSpy.mock.calls[0];
		expect(url).toBe('/api/settings');
		expect(init?.method).toBe('PUT');
		expect(JSON.parse(String(init?.body))).toEqual({
			data_backup_enabled: 'false',
			data_backup_time: '04:15',
			data_backup_retention_count: '7'
		});
		await expect.element(page.getByRole('button', { name: 'Save' })).not.toBeInTheDocument();
	});

	it('edits a URL extra with a URL input', async () => {
		const fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValue(new Response(JSON.stringify({ message: 'ok' }), { status: 200 }));
		await render(ScheduledTaskCard, {
			task: task({
				name: 'bulk_data_update',
				display_name: 'Card Data Update',
				enabled_key: 'bulk_data_update_enabled',
				time_key: 'bulk_data_update_time',
				extras: [
					{
						key: 'bulk_data_url',
						label: 'Bulk data source',
						description: 'Scryfall bulk data API endpoint.',
						kind: 'url',
						value: 'http://127.0.0.1:9/'
					}
				]
			}),
			onUpdated: vi.fn()
		});

		await page.getByRole('button', { name: 'Edit schedule' }).click();
		const input = page.getByLabelText(/^Bulk data source/);
		await expect.element(input).toHaveAttribute('type', 'url');
		await expect.element(input).toHaveValue('http://127.0.0.1:9/');
		await input.fill('https://api.scryfall.com/bulk-data');
		await page.getByRole('button', { name: 'Save' }).click();

		await vi.waitFor(() => expect(fetchSpy).toHaveBeenCalled());
		expect(JSON.parse(String(fetchSpy.mock.calls[0][1]?.body))).toMatchObject({
			bulk_data_url: 'https://api.scryfall.com/bulk-data'
		});
	});

	it('keeps the form open and shows the validation error when saving fails', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			new Response(JSON.stringify({ error: 'invalid setting value: data_backup_time' }), {
				status: 400
			})
		);
		const error = vi.spyOn(notifications, 'error');
		const onUpdated = vi.fn();
		await render(ScheduledTaskCard, { task: task(), onUpdated });

		await page.getByRole('button', { name: 'Edit schedule' }).click();
		await page.getByRole('button', { name: 'Save' }).click();

		await vi.waitFor(() =>
			expect(error).toHaveBeenCalledWith('invalid setting value: data_backup_time')
		);
		expect(onUpdated).not.toHaveBeenCalled();
		await expect.element(page.getByRole('button', { name: 'Save' })).toBeInTheDocument();
	});

	it('discards edits on cancel', async () => {
		const fetchSpy = vi.spyOn(globalThis, 'fetch');
		await render(ScheduledTaskCard, { task: task() });

		await page.getByRole('button', { name: 'Edit schedule' }).click();
		await page.getByLabelText(/^Time of day/).fill('09:00');
		await page.getByRole('button', { name: 'Cancel' }).click();

		await expect.element(page.getByLabelText(/^Time of day/)).not.toBeInTheDocument();
		expect(fetchSpy).not.toHaveBeenCalled();
	});
});
