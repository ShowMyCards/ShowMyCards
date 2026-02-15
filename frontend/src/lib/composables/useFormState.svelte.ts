/**
 * Reusable form state management with Svelte 5 runes
 *
 * Manages form values, validation errors, and submission state.
 *
 * @example
 * ```ts
 * const form = useFormState({
 *   name: '',
 *   email: ''
 * });
 *
 * // Update a field
 * form.setField('name', 'John Doe');
 *
 * // Submit the form
 * await form.submit(async (values) => {
 *   await api.create(values);
 * }, {
 *   onSuccess: () => console.log('Created!'),
 *   resetOnSuccess: true
 * });
 * ```
 */
export function useFormState<T extends Record<string, unknown>>(initialValues: T) {
	let values = $state<T>(structuredClone(initialValues));
	let errors = $state<Partial<Record<keyof T, string>>>({});
	let isSubmitting = $state(false);

	/**
	 * Reset form to initial values and clear errors
	 */
	function reset() {
		values = structuredClone(initialValues);
		errors = {};
	}

	/**
	 * Update a single field value and clear its error
	 */
	function setField<K extends keyof T>(field: K, value: T[K]) {
		values[field] = value;
		if (errors[field]) {
			errors[field] = undefined;
		}
	}

	/**
	 * Set an error message for a field
	 */
	function setError<K extends keyof T>(field: K, message: string) {
		errors[field] = message;
	}

	/**
	 * Clear an error for a field
	 */
	function clearError<K extends keyof T>(field: K) {
		errors[field] = undefined;
	}

	/**
	 * Submit the form
	 *
	 * @param action - Async function to execute with form values
	 * @param options - Submission options
	 */
	async function submit(
		action: (values: T) => Promise<void>,
		options?: {
			onSuccess?: () => void;
			onError?: (error: Error) => void;
			resetOnSuccess?: boolean;
		}
	) {
		isSubmitting = true;
		errors = {};

		try {
			await action(values);

			if (options?.resetOnSuccess !== false) {
				reset();
			}

			options?.onSuccess?.();
		} catch (error) {
			options?.onError?.(error as Error);
			throw error;
		} finally {
			isSubmitting = false;
		}
	}

	return {
		get values() {
			return values;
		},
		get errors() {
			return errors;
		},
		get isSubmitting() {
			return isSubmitting;
		},
		get hasErrors() {
			return Object.keys(errors).length > 0;
		},
		setField,
		setError,
		clearError,
		reset,
		submit
	};
}
