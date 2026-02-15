import { BACKEND_URL } from '$lib';
import type { PageServerLoad, Actions } from './$types';
import { fail } from '@sveltejs/kit';

export const load: PageServerLoad = async ({ fetch }) => {
	try {
		const response = await fetch(`${BACKEND_URL}/api/settings`);

		if (!response.ok) {
			return {
				settings: {},
				error: 'Failed to load settings'
			};
		}

		const settings = await response.json();

		return {
			settings,
			error: null
		};
	} catch {
		return {
			settings: {},
			error: 'Failed to load settings'
		};
	}
};

export const actions = {
	// Save settings
	save: async ({ request, fetch }) => {
		const data = await request.formData();
		const settingsJson = data.get('settings') as string;

		if (!settingsJson) {
			return fail(400, { error: 'Settings data is required' });
		}

		try {
			const settings = JSON.parse(settingsJson);

			const response = await fetch(`${BACKEND_URL}/api/settings`, {
				method: 'PUT',
				headers: {
					'Content-Type': 'application/json'
				},
				body: JSON.stringify(settings)
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, {
					error: errorData.error || 'Failed to save settings'
				});
			}

			return {
				success: true,
				action: 'save'
			};
		} catch {
			return fail(500, { error: 'Failed to save settings' });
		}
	},

	// Trigger bulk data import
	triggerImport: async ({ fetch }) => {
		try {
			const response = await fetch(`${BACKEND_URL}/api/bulk-data/import`, {
				method: 'POST'
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				return fail(response.status, {
					error: errorData.error || 'Failed to trigger import'
				});
			}

			const result = await response.json();
			return {
				success: true,
				action: 'import',
				job_id: result.job_id,
				message: `Import started (Job ID: ${result.job_id})`
			};
		} catch {
			return fail(500, { error: 'Failed to trigger import' });
		}
	}
} satisfies Actions;
