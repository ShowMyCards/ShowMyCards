import { TIMEOUTS } from '../constants';

/**
 * Reusable debounced expression validation
 *
 * Validates sorting rule expressions against the backend API with automatic
 * debouncing to reduce unnecessary API calls.
 *
 * @example
 * ```ts
 * const validation = useExpressionValidation({ debounce: 500 });
 *
 * // Set expression to validate
 * validation.setExpression('rarity == "mythic"');
 *
 * // Check validation state
 * if (validation.isValid) {
 *   console.log('Expression is valid!');
 * } else if (validation.error) {
 *   console.error('Validation error:', validation.error);
 * }
 * ```
 */
export function useExpressionValidation(options?: { debounce?: number }) {
	const debounceMs = options?.debounce ?? TIMEOUTS.VALIDATION_DEBOUNCE;

	let expression = $state('');
	let isValidating = $state(false);
	let validationResult = $state<{
		isValid: boolean;
		error?: string;
	} | null>(null);
	let timeoutId: ReturnType<typeof setTimeout> | null = null;

	/**
	 * Validate the expression against the backend via API proxy route
	 */
	async function validate(expr: string) {
		if (!expr.trim()) {
			validationResult = null;
			return;
		}

		isValidating = true;

		try {
			const response = await fetch('/api/validate-expression', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ expression: expr })
			});

			const data = await response.json();
			validationResult = {
				isValid: data.valid,
				error: data.error
			};
		} catch (e) {
			validationResult = {
				isValid: false,
				error: 'Validation request failed'
			};
		} finally {
			isValidating = false;
		}
	}

	/**
	 * Set the expression to validate (with debouncing)
	 */
	function setExpression(expr: string) {
		expression = expr;
		validationResult = null;

		if (timeoutId) {
			clearTimeout(timeoutId);
		}

		if (expr.trim()) {
			timeoutId = setTimeout(() => validate(expr), debounceMs);
		}
	}

	// Cleanup timeout on unmount
	$effect(() => {
		return () => {
			if (timeoutId) clearTimeout(timeoutId);
		};
	});

	return {
		get expression() {
			return expression;
		},
		get isValidating() {
			return isValidating;
		},
		get validationResult() {
			return validationResult;
		},
		get isValid() {
			return validationResult?.isValid ?? false;
		},
		get error() {
			return validationResult?.error;
		},
		setExpression
	};
}
