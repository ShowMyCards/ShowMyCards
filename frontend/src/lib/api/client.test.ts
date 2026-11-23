import { afterEach, describe, expect, it, vi } from 'vitest';
import { ApiError, apiClient } from './client';

describe('apiClient errors', () => {
	afterEach(() => vi.restoreAllMocks());

	it("uses the backend's `error` field as the message", async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			new Response(JSON.stringify({ error: 'data_backup_time must be a time of day (HH:MM)' }), {
				status: 400
			})
		);

		const request = apiClient.put('/settings', { data_backup_time: '25:00' });

		await expect(request).rejects.toBeInstanceOf(ApiError);
		await expect(request).rejects.toThrow('data_backup_time must be a time of day (HH:MM)');
	});

	it('falls back to the HTTP status when the body has no message', async () => {
		vi.spyOn(globalThis, 'fetch').mockResolvedValue(
			new Response('', { status: 500, statusText: 'Internal Server Error' })
		);

		await expect(apiClient.get('/settings')).rejects.toThrow('HTTP 500: Internal Server Error');
	});
});
