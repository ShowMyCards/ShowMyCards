import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { JobRecord } from '#lib';
import JobDetailModal from './JobDetailModal.svelte';

function job(overrides: Partial<JobRecord> = {}): JobRecord {
	return {
		id: 12,
		created_at: '2026-11-20T03:30:00Z',
		updated_at: '2026-11-20T03:30:05Z',
		type: 'data_backup',
		status: 'completed',
		started_at: '2026-11-20T03:30:00Z',
		completed_at: '2026-11-20T03:30:05Z',
		...overrides
	};
}

describe('JobDetailModal', () => {
	it("surfaces a backup's file and size, and renders its metadata generically", async () => {
		await render(JobDetailModal, {
			job: job({
				metadata: JSON.stringify({
					file_name: 'showmycards-backup-2026-11-20T033000Z.json.gz',
					size_bytes: 2048,
					retention: 14,
					pruned: 1
				})
			}),
			typeLabel: 'Data Backup',
			onClose: vi.fn()
		});

		await expect.element(page.getByText('Job #12')).toBeInTheDocument();
		await expect.element(page.getByText('Data Backup')).toBeInTheDocument();
		await expect.element(page.getByText('Completed')).toBeInTheDocument();
		await expect.element(page.getByText('5s')).toBeInTheDocument();

		const output = page.getByRole('region', { name: 'Output' });
		await expect
			.element(output.getByText('showmycards-backup-2026-11-20T033000Z.json.gz'))
			.toBeInTheDocument();
		await expect.element(output.getByText('(2.0 KB)')).toBeInTheDocument();

		const details = page.getByRole('region', { name: 'Details' });
		await expect.element(details.getByText('Retention')).toBeInTheDocument();
		await expect.element(details.getByText('Size')).toBeInTheDocument();
	});

	it('shows the error of a failed job and omits empty sections', async () => {
		await render(JobDetailModal, {
			job: job({
				type: 'job_cleanup',
				status: 'failed',
				error: 'database is locked',
				metadata: '{}'
			}),
			typeLabel: 'Job History Cleanup',
			onClose: vi.fn()
		});

		await expect
			.element(page.getByRole('region', { name: 'Error' }).getByText('database is locked'))
			.toBeInTheDocument();
		await expect.element(page.getByRole('region', { name: 'Output' })).not.toBeInTheDocument();
		await expect.element(page.getByRole('region', { name: 'Details' })).not.toBeInTheDocument();
	});

	it('closes', async () => {
		const onClose = vi.fn();
		await render(JobDetailModal, { job: job(), typeLabel: 'Data Backup', onClose });

		await page.getByRole('button', { name: 'Close', exact: true }).click();

		expect(onClose).toHaveBeenCalled();
	});

	it('shows deck re-sort counts on a completed bulk import', async () => {
		await render(JobDetailModal, {
			job: job({
				type: 'bulk_data_import',
				status: 'completed',
				metadata: JSON.stringify({
					total_cards: 100,
					processed_cards: 100,
					failed_cards: 0,
					phase: 'completed',
					deck_items_checked: 14,
					deck_items_moved: 3
				})
			}),
			typeLabel: 'Card Data Update',
			onClose: vi.fn()
		});

		await expect.element(page.getByText('Completed', { exact: true })).toBeInTheDocument();
		const details = page.getByRole('region', { name: 'Details' });
		await expect.element(details.getByText('Deck Items Checked')).toBeInTheDocument();
		await expect.element(details.getByText('14', { exact: true })).toBeInTheDocument();
		await expect.element(details.getByText('Deck Items Moved')).toBeInTheDocument();
		await expect.element(details.getByText('3', { exact: true })).toBeInTheDocument();
	});

	it('highlights a non-fatal deck re-sort error on a completed job', async () => {
		await render(JobDetailModal, {
			job: job({
				type: 'bulk_data_import',
				status: 'completed',
				metadata: JSON.stringify({
					total_cards: 100,
					processed_cards: 100,
					failed_cards: 0,
					phase: 'completed',
					deck_resort_error: 'loading deck items: database is locked'
				})
			}),
			typeLabel: 'Card Data Update',
			onClose: vi.fn()
		});

		await expect.element(page.getByText('Completed', { exact: true })).toBeInTheDocument();
		await expect.element(page.getByRole('region', { name: 'Error' })).not.toBeInTheDocument();

		const details = page.getByRole('region', { name: 'Details' });
		const label = details.getByText('Deck Resort Error');
		const message = details.getByText('loading deck items: database is locked');
		await expect.element(label).toHaveClass('text-error');
		await expect.element(message).toHaveClass('text-error');
	});

	it('renders nothing without a job', async () => {
		await render(JobDetailModal, { job: null, typeLabel: '', onClose: vi.fn() });

		await expect.element(page.getByText('Created')).not.toBeInTheDocument();
	});
});
