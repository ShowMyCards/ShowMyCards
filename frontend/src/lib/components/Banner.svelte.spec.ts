import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { Banner as BannerData } from '$lib';
import Banner from './Banner.svelte';

function makeBanner(overrides: Partial<BannerData> = {}): BannerData {
	return {
		id: 'sync:bulk_data_import',
		severity: 'info',
		message: 'Updating card data from Scryfall.',
		dismissible: false,
		...overrides
	};
}

describe('Banner', () => {
	it('renders the message', async () => {
		render(Banner, { banner: makeBanner({ message: 'Sync in progress' }) });

		await expect.element(page.getByText('Sync in progress')).toBeInTheDocument();
	});

	it('announces politely for info severity', async () => {
		render(Banner, { banner: makeBanner({ severity: 'info' }) });

		await expect.element(page.getByRole('status')).toBeInTheDocument();
	});

	it('announces assertively and styles for warning severity', async () => {
		render(Banner, { banner: makeBanner({ severity: 'warning' }) });

		await expect.element(page.getByRole('alert')).toHaveClass('alert-warning');
	});

	it('shows a dismiss button when the banner is dismissible', async () => {
		render(Banner, { banner: makeBanner({ dismissible: true }) });

		await expect.element(page.getByRole('button', { name: /dismiss/i })).toBeInTheDocument();
	});

	it('hides the dismiss button when the banner is not dismissible', async () => {
		render(Banner, { banner: makeBanner({ dismissible: false }) });

		await expect.element(page.getByRole('button', { name: /dismiss/i })).not.toBeInTheDocument();
	});

	it('calls ondismiss with the banner id when dismissed', async () => {
		const ondismiss = vi.fn();
		render(Banner, {
			banner: makeBanner({ id: 'job-failed:set_data_import:5', dismissible: true }),
			ondismiss
		});

		await page.getByRole('button', { name: /dismiss/i }).click();

		expect(ondismiss).toHaveBeenCalledWith('job-failed:set_data_import:5');
	});

	it('renders the call-to-action link when one is provided', async () => {
		render(Banner, {
			banner: makeBanner({ link: { label: 'View jobs', href: '/jobs' } })
		});

		await expect
			.element(page.getByRole('link', { name: 'View jobs' }))
			.toHaveAttribute('href', '/jobs');
	});
});
