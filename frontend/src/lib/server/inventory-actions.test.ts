import { describe, it, expect, vi, type Mock } from 'vitest';
import { handleUpdateInventory } from './inventory-actions';

const BACKEND_URL = 'http://localhost:3000';

function makeRequest(fields: Record<string, string>): Request {
	const formData = new FormData();
	for (const [key, value] of Object.entries(fields)) {
		formData.append(key, value);
	}
	return new Request('http://localhost/', { method: 'POST', body: formData });
}

function mockFetch(response: Response = new Response(null, { status: 200 })): Mock {
	return vi.fn().mockResolvedValue(response);
}

function asFetch(fetchMock: Mock): typeof globalThis.fetch {
	return fetchMock as unknown as typeof globalThis.fetch;
}

describe('handleUpdateInventory', () => {
	it('sends a PUT to the inventory item with the new quantity', async () => {
		const fetchMock = mockFetch();

		await handleUpdateInventory(
			makeRequest({ inventory_id: '42', quantity: '3' }),
			asFetch(fetchMock)
		);

		expect(fetchMock).toHaveBeenCalledTimes(1);
		const url = fetchMock.mock.calls[0][0] as string;
		const init = fetchMock.mock.calls[0][1] as RequestInit;
		expect(url).toBe(`${BACKEND_URL}/inventory/42`);
		expect(init.method).toBe('PUT');
		expect(JSON.parse(init.body as string)).toEqual({ quantity: 3 });
	});

	it('returns a success result when the backend accepts the update', async () => {
		const result = await handleUpdateInventory(
			makeRequest({ inventory_id: '42', quantity: '3' }),
			asFetch(mockFetch())
		);

		expect(result).toEqual({ success: true, action: 'update' });
	});

	it('fails with 400 when inventory_id is missing', async () => {
		const fetchMock = mockFetch();

		const result = await handleUpdateInventory(makeRequest({ quantity: '3' }), asFetch(fetchMock));

		expect(result).toMatchObject({ status: 400 });
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('fails with 400 when quantity is missing', async () => {
		const fetchMock = mockFetch();

		const result = await handleUpdateInventory(
			makeRequest({ inventory_id: '42' }),
			asFetch(fetchMock)
		);

		expect(result).toMatchObject({ status: 400 });
		expect(fetchMock).not.toHaveBeenCalled();
	});

	it('propagates the backend status when the update is rejected', async () => {
		const result = await handleUpdateInventory(
			makeRequest({ inventory_id: '42', quantity: '3' }),
			asFetch(mockFetch(new Response(null, { status: 404 })))
		);

		expect(result).toMatchObject({ status: 404 });
	});

	it('fails with 500 when the backend request throws', async () => {
		const fetchMock = vi.fn().mockRejectedValue(new Error('network down'));

		const result = await handleUpdateInventory(
			makeRequest({ inventory_id: '42', quantity: '3' }),
			asFetch(fetchMock)
		);

		expect(result).toMatchObject({ status: 500 });
	});
});
