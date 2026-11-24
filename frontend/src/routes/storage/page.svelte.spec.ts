import { page } from 'vitest/browser';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';
import { refreshAll } from '$app/navigation';
import { notifications, type StorageLocation } from '#lib';
import Page from './+page.svelte';
import type { PageData } from './$types';

// Kit's own runtime imports afterNavigate from this module, so it must be stubbed too.
vi.mock('$app/navigation', () => ({ refreshAll: vi.fn(), afterNavigate: vi.fn() }));
// SvelteKit 3's `deserialize` delegates `parsed.data` to `#app/internal/transport`'s
// `parse()`, which throws `called parse before init_transport` in Vitest browser mode
// because Kit's client bootstrap never runs.
vi.mock('$app/forms', () => ({
	applyAction: vi.fn(),
	enhance: () => ({ destroy() {} }),
	deserialize: (text: string) => JSON.parse(text)
}));

function makeLocation(
	id: number,
	name: string,
	sortOrder: number,
	storageType: 'Box' | 'Binder' = 'Box'
): StorageLocation {
	return {
		id,
		created_at: '2026-10-08T10:00:00Z',
		updated_at: '2026-10-08T10:00:00Z',
		name,
		storage_type: storageType,
		sort_order: sortOrder
	};
}

function makeData(locations: StorageLocation[], error: string | null = null): PageData {
	return { locations, error } as PageData;
}

function renderedLocationNames(container: HTMLElement): string[] {
	return Array.from(container.querySelectorAll('.font-semibold')).map(
		(el) => el.textContent?.trim() ?? ''
	);
}

function successResponse() {
	return new Response(JSON.stringify({ type: 'success', status: 200 }), { status: 200 });
}

function failureResponse(message: string) {
	return new Response(JSON.stringify({ type: 'failure', status: 500, data: { error: message } }), {
		status: 500
	});
}

function parseReorderPayload(
	call: Parameters<typeof fetch>
): Array<{ id: number; sort_order: number }> {
	const formData = call[1]?.body as FormData;
	return JSON.parse(String(formData.get('updates')));
}

function deferred<T = void>() {
	let resolve!: (value: T | PromiseLike<T>) => void;
	let reject!: (reason?: unknown) => void;
	const promise = new Promise<T>((res, rej) => {
		resolve = res;
		reject = rej;
	});
	return { promise, resolve, reject };
}

describe('/storage/+page.svelte', () => {
	afterEach(() => {
		vi.restoreAllMocks();
		vi.mocked(refreshAll).mockReset();
		notifications.clear();
	});

	it('reflects server data updates directly without effect syncing', async () => {
		const initial = [makeLocation(1, 'Alpha Box', 1), makeLocation(2, 'Beta Binder', 2, 'Binder')];
		const screen = await render(Page, { data: makeData(initial) });

		expect(renderedLocationNames(screen.container)).toEqual(['Alpha Box', 'Beta Binder']);

		await screen.rerender({
			data: makeData([...initial, makeLocation(3, 'Gamma Box', 3)])
		});

		expect(renderedLocationNames(screen.container)).toEqual([
			'Alpha Box',
			'Beta Binder',
			'Gamma Box'
		]);
	});

	it('applies optimistic reorder immediately and persists the new sort order', async () => {
		const initial = [
			makeLocation(1, 'Alpha Box', 1),
			makeLocation(2, 'Beta Box', 2),
			makeLocation(3, 'Gamma Box', 3)
		];
		const saveRequest = deferred<Response>();
		const fetchSpy = vi.spyOn(globalThis, 'fetch').mockReturnValueOnce(saveRequest.promise);

		const screen = await render(Page, { data: makeData(initial) });

		vi.mocked(refreshAll).mockImplementationOnce(async () => {
			await screen.rerender({
				data: makeData([
					makeLocation(2, 'Beta Box', 1),
					makeLocation(1, 'Alpha Box', 2),
					makeLocation(3, 'Gamma Box', 3)
				])
			});
		});

		await page.getByRole('button', { name: 'Move Alpha Box down' }).click();

		// Optimistically reordered before fetch resolves
		await vi.waitFor(() =>
			expect(renderedLocationNames(screen.container)).toEqual([
				'Beta Box',
				'Alpha Box',
				'Gamma Box'
			])
		);

		saveRequest.resolve(successResponse());

		await vi.waitFor(() => expect(refreshAll).toHaveBeenCalledTimes(1));
		expect(parseReorderPayload(fetchSpy.mock.calls[0])).toEqual([
			{ id: 2, sort_order: 1 },
			{ id: 1, sort_order: 2 },
			{ id: 3, sort_order: 3 }
		]);
		expect(renderedLocationNames(screen.container)).toEqual(['Beta Box', 'Alpha Box', 'Gamma Box']);
	});

	it('reverts to server order and notifies when reorder fails', async () => {
		const initial = [makeLocation(1, 'Alpha Box', 1), makeLocation(2, 'Beta Box', 2)];
		vi.spyOn(globalThis, 'fetch').mockResolvedValueOnce(failureResponse('Database locked'));
		const errorSpy = vi.spyOn(notifications, 'error');
		vi.mocked(refreshAll).mockRejectedValueOnce(new Error('offline'));

		const screen = await render(Page, { data: makeData(initial) });

		await page.getByRole('button', { name: 'Move Alpha Box down' }).click();

		await vi.waitFor(() => expect(errorSpy).toHaveBeenCalledWith('Database locked'));
		// Even when refreshAll rejects and `data` never changes, optimistic state clears cleanly
		await vi.waitFor(() =>
			expect(renderedLocationNames(screen.container)).toEqual(['Alpha Box', 'Beta Box'])
		);
	});

	it('reverts to server order on network error', async () => {
		const initial = [makeLocation(1, 'Alpha Box', 1), makeLocation(2, 'Beta Box', 2)];
		vi.spyOn(globalThis, 'fetch').mockRejectedValueOnce(new Error('network down'));
		const errorSpy = vi.spyOn(notifications, 'error');
		vi.mocked(refreshAll).mockRejectedValueOnce(new Error('network down'));

		const screen = await render(Page, { data: makeData(initial) });

		await page.getByRole('button', { name: 'Move Alpha Box down' }).click();

		await vi.waitFor(() =>
			expect(errorSpy).toHaveBeenCalledWith('Failed to save storage location order')
		);
		await vi.waitFor(() =>
			expect(renderedLocationNames(screen.container)).toEqual(['Alpha Box', 'Beta Box'])
		);
	});

	it('keeps optimistic order and persists a reorder queued during refreshAll', async () => {
		const initial = [
			makeLocation(1, 'Alpha Box', 1),
			makeLocation(2, 'Beta Box', 2),
			makeLocation(3, 'Gamma Box', 3)
		];
		const firstRefresh = deferred<void>();
		const secondSave = deferred<Response>();
		const fetchSpy = vi
			.spyOn(globalThis, 'fetch')
			.mockResolvedValueOnce(successResponse())
			.mockReturnValueOnce(secondSave.promise);

		const screen = await render(Page, { data: makeData(initial) });

		vi.mocked(refreshAll)
			.mockImplementationOnce(async () => {
				await firstRefresh.promise;
				await screen.rerender({
					data: makeData([
						makeLocation(2, 'Beta Box', 1),
						makeLocation(1, 'Alpha Box', 2),
						makeLocation(3, 'Gamma Box', 3)
					])
				});
			})
			.mockImplementationOnce(async () => {
				await screen.rerender({
					data: makeData([
						makeLocation(2, 'Beta Box', 1),
						makeLocation(3, 'Gamma Box', 2),
						makeLocation(1, 'Alpha Box', 3)
					])
				});
			});

		// First move completes its fetch and pauses inside refreshAll
		await page.getByRole('button', { name: 'Move Alpha Box down' }).click();
		await vi.waitFor(() => expect(refreshAll).toHaveBeenCalledTimes(1));

		// Second move arrives while refreshAll for the first move is still in flight
		await page.getByRole('button', { name: 'Move Alpha Box down' }).click();
		await vi.waitFor(() =>
			expect(renderedLocationNames(screen.container)).toEqual([
				'Beta Box',
				'Gamma Box',
				'Alpha Box'
			])
		);

		// First refresh resolves with the older order; optimistic overlay must not snap back
		firstRefresh.resolve();
		await vi.waitFor(() => expect(fetchSpy).toHaveBeenCalledTimes(2));
		expect(renderedLocationNames(screen.container)).toEqual(['Beta Box', 'Gamma Box', 'Alpha Box']);
		expect(parseReorderPayload(fetchSpy.mock.calls[1])).toEqual([
			{ id: 2, sort_order: 1 },
			{ id: 3, sort_order: 2 },
			{ id: 1, sort_order: 3 }
		]);

		secondSave.resolve(successResponse());
		await vi.waitFor(() => expect(refreshAll).toHaveBeenCalledTimes(2));
		expect(renderedLocationNames(screen.container)).toEqual(['Beta Box', 'Gamma Box', 'Alpha Box']);
	});
});
