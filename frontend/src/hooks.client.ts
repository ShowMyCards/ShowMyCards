import type { HandleClientError } from '@sveltejs/kit';

/**
 * Global client-side error handler
 * Catches unhandled errors and provides better error reporting
 */
export const handleError: HandleClientError = ({ error, event, status, message }) => {
	// Log error to console in development
	if (import.meta.env.DEV) {
		console.error('Client error:', {
			error,
			event: event.url.pathname,
			status,
			message
		});
	}

	// In production, you could send errors to an error tracking service
	// Example: Sentry, LogRocket, etc.
	// if (import.meta.env.PROD) {
	//   reportErrorToService({ error, event, status, message });
	// }

	// Provide user-friendly error messages
	const userMessage = getUserFriendlyMessage(status, message, error);

	return {
		message: userMessage,
		status
	};
};

/**
 * Convert technical error messages to user-friendly ones
 */
function getUserFriendlyMessage(status: number, message: string, error: unknown): string {
	// For network errors
	if (error instanceof TypeError && error.message.includes('fetch')) {
		return 'Unable to connect to the server. Please check your internet connection.';
	}

	// For specific status codes
	switch (status) {
		case 400:
			return 'Invalid request. Please check your input and try again.';
		case 401:
			return 'You need to be logged in to access this resource.';
		case 403:
			return 'You do not have permission to access this resource.';
		case 404:
			return 'The requested resource was not found.';
		case 408:
			return 'Request timeout. Please try again.';
		case 429:
			return 'Too many requests. Please wait a moment and try again.';
		case 500:
			return 'An internal server error occurred. Please try again later.';
		case 502:
			return 'Bad gateway. The server is temporarily unavailable.';
		case 503:
			return 'Service unavailable. Please try again in a few moments.';
		case 504:
			return 'Gateway timeout. The request took too long to process.';
		default:
			// Return the original message if it's user-friendly, otherwise provide a generic message
			return message || 'An unexpected error occurred. Please try again.';
	}
}
