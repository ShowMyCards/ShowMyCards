import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import Page from './+page.svelte';
import type { PageData } from './$types';

// Kit's own runtime (e.g. $app/forms) imports afterNavigate from this module, so it must be stubbed too.
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn(), afterNavigate: vi.fn() }));

function makeData(): PageData {
	return {
		settings: {
			scryfall_default_search: 'game:paper',
			scryfall_unique_mode: 'cards',
			scryfall_default_language: 'en',
			bulk_data_update_enabled: 'true',
			bulk_data_update_time: '03:00',
			data_backup_last_run: '2026-11-20T03:30:00Z'
		},
		error: null
	} as PageData;
}

describe('/settings/+page.svelte', () => {
	it('no longer renders schedule controls and points to the Jobs page', async () => {
		await render(Page, { data: makeData() });

		await expect.element(page.getByText('Scryfall Search Options')).toBeInTheDocument();
		await expect.element(page.getByText('Automatic Updates')).not.toBeInTheDocument();
		await expect.element(page.getByText('Update Time')).not.toBeInTheDocument();
		await expect.element(page.getByRole('link', { name: 'Jobs page' })).toBeInTheDocument();
	});

	it('submits only the Scryfall keys it edits', async () => {
		const { container } = await render(Page, { data: makeData() });

		const hidden = container.querySelector<HTMLInputElement>(
			'input[type="hidden"][name="settings"]'
		);
		expect(JSON.parse(hidden?.value ?? '{}')).toEqual({
			scryfall_default_search: 'game:paper',
			scryfall_unique_mode: 'cards',
			scryfall_default_language: 'en'
		});
	});
});
