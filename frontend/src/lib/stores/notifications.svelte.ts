import { TIMEOUTS } from '../constants';

/**
 * Notification message data structure
 */
export interface NotificationMessage {
	id: string;
	type: 'success' | 'error' | 'info' | 'warning';
	message: string;
}

/**
 * Generate a unique ID with fallback for environments without crypto.randomUUID
 */
function generateId(): string {
	if (typeof crypto !== 'undefined' && crypto.randomUUID) {
		return crypto.randomUUID();
	}
	// Fallback: generate a simple unique ID
	return `${Date.now()}-${Math.random().toString(36).substring(2, 9)}`;
}

/**
 * Centralized notification store using Svelte 5 runes
 *
 * Manages application-wide notifications with automatic dismissal.
 *
 * @example
 * ```ts
 * import { notifications } from '$lib';
 *
 * // Show a success notification
 * notifications.success('Item created successfully!');
 *
 * // Show an error with custom duration
 * notifications.error('Failed to save', 10000);
 *
 * // Manually dismiss a notification
 * notifications.dismiss(notificationId);
 * ```
 */
class NotificationStore {
	notifications = $state<NotificationMessage[]>([]);

	/**
	 * Show a notification with the specified type and message
	 *
	 * @param type - Notification type (success, error, info, warning)
	 * @param message - Message to display
	 * @param duration - Auto-dismiss duration in ms (0 to disable auto-dismiss)
	 */
	show(
		type: 'success' | 'error' | 'info' | 'warning',
		message: string,
		duration: number = TIMEOUTS.NOTIFICATION_DISMISS
	) {
		const id = generateId();
		this.notifications.push({ id, type, message });

		if (duration > 0) {
			setTimeout(() => this.dismiss(id), duration);
		}
	}

	/**
	 * Show a success notification
	 *
	 * @param message - Success message
	 * @param duration - Auto-dismiss duration in ms
	 */
	success(message: string, duration?: number) {
		this.show('success', message, duration);
	}

	/**
	 * Show an error notification
	 *
	 * @param message - Error message
	 * @param duration - Auto-dismiss duration in ms
	 */
	error(message: string, duration?: number) {
		this.show('error', message, duration);
	}

	/**
	 * Show an info notification
	 *
	 * @param message - Info message
	 * @param duration - Auto-dismiss duration in ms
	 */
	info(message: string, duration?: number) {
		this.show('info', message, duration);
	}

	/**
	 * Show a warning notification
	 *
	 * @param message - Warning message
	 * @param duration - Auto-dismiss duration in ms
	 */
	warning(message: string, duration?: number) {
		this.show('warning', message, duration);
	}

	/**
	 * Manually dismiss a notification by ID
	 *
	 * @param id - Notification ID to dismiss
	 */
	dismiss(id: string) {
		this.notifications = this.notifications.filter((n) => n.id !== id);
	}

	/**
	 * Clear all notifications
	 */
	clear() {
		this.notifications = [];
	}
}

/**
 * Global notification store instance
 */
export const notifications = new NotificationStore();
