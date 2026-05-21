import { page } from 'vitest/browser';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import type { Banner } from '$lib';

const listMock = vi.hoisted(() => vi.fn());

vi.mock('$lib/api/resources/banners', () => ({
	bannersApi: { list: listMock }
}));

// SvelteKit's dynamic public env isn't wired up in the component test
// environment; stub it so the module graph imports cleanly.
vi.mock('$env/dynamic/public', () => ({ env: {} }));

import Banners from './Banners.svelte';

function makeBanner(overrides: Partial<Banner> = {}): Banner {
	return {
		id: 'sync:bulk_data_import',
		severity: 'info',
		message: 'Updating card data from Scryfall.',
		dismissible: false,
		...overrides
	};
}

describe('Banners', () => {
	beforeEach(() => {
		listMock.mockReset();
		localStorage.clear();
	});

	it('renders nothing when the API returns no banners', async () => {
		listMock.mockResolvedValue([]);
		render(Banners);

		await vi.waitFor(() => expect(listMock).toHaveBeenCalled());
		await expect.element(page.getByRole('status')).not.toBeInTheDocument();
	});

	it('renders a banner returned by the API', async () => {
		listMock.mockResolvedValue([makeBanner({ message: 'Sync running' })]);
		render(Banners);

		await expect.element(page.getByText('Sync running')).toBeInTheDocument();
	});

	it('hides a banner once dismissed and persists the dismissal', async () => {
		listMock.mockResolvedValue([
			makeBanner({ id: 'job-failed:bulk_data_import:7', message: 'Sync failed', dismissible: true })
		]);
		render(Banners);

		await expect.element(page.getByText('Sync failed')).toBeInTheDocument();
		await page.getByRole('button', { name: /dismiss/i }).click();
		await expect.element(page.getByText('Sync failed')).not.toBeInTheDocument();

		const stored = JSON.parse(localStorage.getItem('dismissedBanners') ?? '[]');
		expect(stored).toContain('job-failed:bulk_data_import:7');
	});

	it('does not show a banner whose id was previously dismissed', async () => {
		localStorage.setItem('dismissedBanners', JSON.stringify(['job-failed:bulk_data_import:7']));
		listMock.mockResolvedValue([
			makeBanner({ id: 'job-failed:bulk_data_import:7', message: 'Sync failed', dismissible: true })
		]);
		render(Banners);

		await vi.waitFor(() => expect(listMock).toHaveBeenCalled());
		await expect.element(page.getByText('Sync failed')).not.toBeInTheDocument();
	});
});
