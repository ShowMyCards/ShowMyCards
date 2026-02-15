import { notifications } from '../stores/notifications.svelte';

/**
 * Reusable async action handler with loading and error states
 *
 * Wraps async operations with automatic loading state tracking, error handling,
 * and optional notification integration.
 *
 * @example
 * ```ts
 * const deleteAction = useAsyncAction(
 *   async (id: number) => {
 *     await storageApi.delete(id);
 *   },
 *   {
 *     onSuccess: 'Storage location deleted successfully',
 *     onError: 'Failed to delete storage location'
 *   }
 * );
 *
 * // Execute the action
 * await deleteAction.execute(123);
 * ```
 */
export function useAsyncAction<TArgs extends unknown[]>(
	action: (...args: TArgs) => Promise<void>,
	options?: {
		onSuccess?: string | (() => void);
		onError?: string | ((error: Error) => void);
	}
) {
	let isLoading = $state(false);
	let error = $state<string | null>(null);

	/**
	 * Execute the async action
	 *
	 * @param args - Arguments to pass to the action
	 */
	async function execute(...args: TArgs) {
		isLoading = true;
		error = null;

		try {
			await action(...args);

			// Handle success callback/notification
			if (options?.onSuccess) {
				if (typeof options.onSuccess === 'string') {
					notifications.success(options.onSuccess);
				} else {
					options.onSuccess();
				}
			}
		} catch (e) {
			const errorMessage = e instanceof Error ? e.message : 'An error occurred';
			error = errorMessage;

			// Handle error callback/notification
			if (options?.onError) {
				if (typeof options.onError === 'string') {
					notifications.error(options.onError);
				} else {
					options.onError(e as Error);
				}
			} else {
				// Default: show error notification
				notifications.error(errorMessage);
			}
		} finally {
			isLoading = false;
		}
	}

	/**
	 * Reset error state
	 */
	function reset() {
		error = null;
	}

	return {
		execute,
		get isLoading() {
			return isLoading;
		},
		get error() {
			return error;
		},
		reset
	};
}
