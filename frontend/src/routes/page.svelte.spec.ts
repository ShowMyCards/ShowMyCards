import { page } from 'vitest/browser';
import { describe, expect, it } from 'vitest';
import { render } from 'vitest-browser-svelte';
import Page from './+page.svelte';
import type { PageData } from './$types';

// +page.svelte expects the shape returned by its server load. Rendering the
// component in isolation means supplying it directly.
const data: PageData = {
	stats: {
		total_inventory_cards: 0,
		total_wishlist_cards: 0,
		total_collection_value: 0,
		total_collected_from_lists: 0,
		total_remaining_lists_value: 0,
		total_storage_locations: 0,
		total_lists: 0,
		unassigned_cards: 0
	}
};

describe('/+page.svelte', () => {
	it('should render h1', async () => {
		render(Page, { data });

		const heading = page.getByRole('heading', { level: 1 });
		await expect.element(heading).toBeInTheDocument();
	});
});
