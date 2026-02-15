import { BACKEND_URL } from '../config';

/**
 * Custom error class for API errors
 */
export class ApiError extends Error {
	constructor(
		message: string,
		public status: number,
		public data?: unknown
	) {
		super(message);
		this.name = 'ApiError';
	}
}

/**
 * Base API client for making HTTP requests
 *
 * Provides type-safe methods for CRUD operations with automatic JSON parsing
 * and error handling.
 *
 * @example
 * ```ts
 * const client = new ApiClient('http://localhost:3000');
 * const data = await client.get<StorageLocation[]>('/storage');
 * ```
 */
export class ApiClient {
	constructor(private baseUrl: string) {}

	/**
	 * Make a request to the API
	 *
	 * @param path - API endpoint path (e.g., '/storage')
	 * @param options - Fetch options
	 * @returns Parsed JSON response
	 * @throws {ApiError} If the request fails
	 */
	private async request<T>(path: string, options?: RequestInit): Promise<T> {
		const url = `${this.baseUrl}${path}`;

		try {
			const response = await fetch(url, {
				...options,
				headers: {
					'Content-Type': 'application/json',
					...options?.headers
				}
			});

			if (!response.ok) {
				const errorData = await response.json().catch(() => ({}));
				throw new ApiError(
					errorData.message || `HTTP ${response.status}: ${response.statusText}`,
					response.status,
					errorData
				);
			}

			// Handle empty responses (e.g., DELETE requests)
			const text = await response.text();
			return text ? JSON.parse(text) : ({} as T);
		} catch (error) {
			// Re-throw ApiError as-is
			if (error instanceof ApiError) {
				throw error;
			}

			// Wrap network errors
			if (error instanceof TypeError) {
				throw new ApiError('Network error: Failed to connect to server', 0, error);
			}

			// Wrap other errors
			throw new ApiError(
				error instanceof Error ? error.message : 'Unknown error occurred',
				0,
				error
			);
		}
	}

	/**
	 * GET request
	 *
	 * @param path - API endpoint path
	 * @returns Parsed JSON response
	 */
	async get<T>(path: string): Promise<T> {
		return this.request<T>(path);
	}

	/**
	 * POST request
	 *
	 * @param path - API endpoint path
	 * @param data - Request body
	 * @returns Parsed JSON response
	 */
	async post<T>(path: string, data: unknown): Promise<T> {
		return this.request<T>(path, {
			method: 'POST',
			body: JSON.stringify(data)
		});
	}

	/**
	 * PUT request
	 *
	 * @param path - API endpoint path
	 * @param data - Request body
	 * @returns Parsed JSON response
	 */
	async put<T>(path: string, data: unknown): Promise<T> {
		return this.request<T>(path, {
			method: 'PUT',
			body: JSON.stringify(data)
		});
	}

	/**
	 * DELETE request
	 *
	 * @param path - API endpoint path
	 * @param data - Optional request body for batch deletions
	 * @returns Parsed JSON response
	 */
	async delete<T = void>(path: string, data?: unknown): Promise<T> {
		return this.request<T>(path, {
			method: 'DELETE',
			body: data ? JSON.stringify(data) : undefined
		});
	}
}

/**
 * Global API client instance
 */
export const apiClient = new ApiClient(BACKEND_URL);
