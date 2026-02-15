import { apiClient } from '../client';
import type { SortingRule, EvaluateRequest, EvaluateResponse } from '$lib';

/**
 * Request type for creating a sorting rule
 */
export interface CreateRuleRequest {
	name: string;
	priority: number;
	expression: string;
	storage_location_id: number;
	enabled?: boolean;
}

/**
 * Request type for updating a sorting rule
 */
export interface UpdateRuleRequest {
	name?: string;
	priority?: number;
	expression?: string;
	storage_location_id?: number;
	enabled?: boolean;
}

/**
 * Request type for validating an expression
 */
export interface ValidateExpressionRequest {
	expression: string;
}

/**
 * Response type for expression validation
 */
export interface ValidateExpressionResponse {
	valid: boolean;
	error?: string;
}

/**
 * Request type for batch updating priorities
 */
export interface BatchUpdatePrioritiesRequest {
	updates: Array<{
		id: number;
		priority: number;
	}>;
}

/**
 * Response type for batch priority updates
 */
export interface BatchUpdatePrioritiesResponse {
	updated_count: number;
}

/**
 * Query parameters for listing rules
 */
export interface ListRulesParams {
	enabled?: boolean;
}

/**
 * Sorting rules API methods
 *
 * Provides type-safe access to sorting rule endpoints.
 *
 * @example
 * ```ts
 * import { rulesApi } from '$lib/api';
 *
 * // List all enabled rules
 * const rules = await rulesApi.list({ enabled: true });
 *
 * // Validate an expression
 * const validation = await rulesApi.validate({ expression: 'rarity == "mythic"' });
 *
 * // Evaluate a card against rules
 * const result = await rulesApi.evaluate({ card_data: cardData });
 * ```
 */
export const rulesApi = {
	/**
	 * List all sorting rules with optional filter
	 */
	list: (params?: ListRulesParams) => {
		const searchParams = new URLSearchParams();
		if (params?.enabled !== undefined) searchParams.set('enabled', String(params.enabled));

		const query = searchParams.toString();
		return apiClient.get<SortingRule[]>(`/sorting-rules${query ? `?${query}` : ''}`);
	},

	/**
	 * Get a single sorting rule by ID
	 */
	get: (id: number) => apiClient.get<SortingRule>(`/sorting-rules/${id}`),

	/**
	 * Create a new sorting rule
	 */
	create: (data: CreateRuleRequest) => apiClient.post<SortingRule>('/sorting-rules', data),

	/**
	 * Update a sorting rule
	 */
	update: (id: number, data: UpdateRuleRequest) =>
		apiClient.put<SortingRule>(`/sorting-rules/${id}`, data),

	/**
	 * Delete a sorting rule
	 */
	delete: (id: number) => apiClient.delete<void>(`/sorting-rules/${id}`),

	/**
	 * Validate an expression
	 */
	validate: (data: ValidateExpressionRequest) =>
		apiClient.post<ValidateExpressionResponse>('/sorting-rules/validate', data),

	/**
	 * Evaluate card data against all enabled rules
	 */
	evaluate: (data: EvaluateRequest) =>
		apiClient.post<EvaluateResponse>('/sorting-rules/evaluate', data),

	/**
	 * Batch update rule priorities
	 */
	batchUpdatePriorities: (
		updates: Array<{ id: number; priority: number }>
	): Promise<BatchUpdatePrioritiesResponse> =>
		apiClient.post<BatchUpdatePrioritiesResponse>('/sorting-rules/batch/priorities', { updates })
};
