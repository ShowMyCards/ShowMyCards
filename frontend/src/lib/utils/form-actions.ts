/**
 * Type-safe accessors for SvelteKit form action result data.
 * Replaces `(result.data as any)?.error` patterns.
 */

/**
 * Extract an error message from a form action failure result.
 */
export function getActionError(
	data: Record<string, unknown> | undefined,
	fallback: string
): string {
	if (data && typeof data === 'object' && typeof data.error === 'string') {
		return data.error;
	}
	return fallback;
}

/**
 * Extract a success message from a form action result.
 */
export function getActionMessage(
	data: Record<string, unknown> | undefined,
	fallback: string
): string {
	if (data && typeof data === 'object' && typeof data.message === 'string') {
		return data.message;
	}
	return fallback;
}
