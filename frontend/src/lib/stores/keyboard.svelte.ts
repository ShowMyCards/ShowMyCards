/**
 * Keyboard shortcuts store with hover target tracking
 *
 * Tracks which element is currently hovered and dispatches keyboard shortcuts to it.
 * Hover over a card, then use Alt+1 to add, Alt+Shift+1 to remove.
 *
 * @example
 * ```svelte
 * <script>
 *   import { keyboard } from '$lib';
 *
 *   // In a CardResultCard component:
 *   const cardActions = {
 *     increment: (treatmentIndex: number) => handleIncrement(treatments[treatmentIndex]),
 *     decrement: (treatmentIndex: number) => handleDecrement(treatments[treatmentIndex]),
 *     treatmentCount: treatments.length
 *   };
 * </script>
 *
 * <div
 *   onmouseenter={() => keyboard.setHoverTarget(card.id, cardActions)}
 *   onmouseleave={() => keyboard.clearHoverTarget(card.id)}
 * >
 *   ...
 * </div>
 * ```
 */

export interface CardActions {
	/** Increment quantity for treatment at index (0-based) */
	increment: (treatmentIndex: number) => void;
	/** Decrement quantity for treatment at index (0-based) */
	decrement: (treatmentIndex: number) => void;
	/** Number of available treatments */
	treatmentCount: number;
}

export interface KeyboardShortcut {
	/** Display key combination (e.g., 'Alt+1', '?') */
	key: string;
	/** Description shown in help overlay */
	description: string;
}

class KeyboardStore {
	/** ID of the currently hovered element */
	hoveredId = $state<string | null>(null);

	/** Actions available on the hovered element */
	hoveredActions = $state<CardActions | null>(null);

	/** Whether the help overlay is visible */
	helpVisible = $state(false);

	/** Whether keyboard shortcuts are enabled */
	enabled = $state(true);

	/** Handle for the global keyboard listener */
	private listenerAttached = false;

	/**
	 * Attach global keyboard listener (call once in root layout)
	 */
	attach() {
		if (this.listenerAttached || typeof window === 'undefined') return;

		window.addEventListener('keydown', this.handleKeydown);
		this.listenerAttached = true;
	}

	/**
	 * Detach global keyboard listener
	 */
	detach() {
		if (!this.listenerAttached || typeof window === 'undefined') return;

		window.removeEventListener('keydown', this.handleKeydown);
		this.listenerAttached = false;
	}

	/**
	 * Handle keydown events
	 */
	private handleKeydown = (event: KeyboardEvent) => {
		if (!this.enabled) return;

		// Ignore if focused on input/textarea/select
		const target = event.target as HTMLElement;
		if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.tagName === 'SELECT') {
			return;
		}

		// Close help on Escape
		if (event.key === 'Escape' && this.helpVisible) {
			this.helpVisible = false;
			event.preventDefault();
			return;
		}

		// Toggle help on ?
		if (event.key === '?') {
			this.helpVisible = !this.helpVisible;
			event.preventDefault();
			return;
		}

		// Alt+1 through Alt+9 for treatment actions
		// Use event.code to handle macOS Alt key producing special characters
		const digitMatch = event.code.match(/^Digit([1-9])$/);
		if (event.altKey && digitMatch) {
			if (!this.hoveredActions) return;

			const treatmentIndex = parseInt(digitMatch[1]) - 1; // Convert to 0-based
			if (treatmentIndex >= this.hoveredActions.treatmentCount) return;

			event.preventDefault();

			if (event.shiftKey) {
				// Alt+Shift+N = decrement
				this.hoveredActions.decrement(treatmentIndex);
			} else {
				// Alt+N = increment
				this.hoveredActions.increment(treatmentIndex);
			}
		}
	};

	/**
	 * Set the currently hovered target
	 */
	setHoverTarget(id: string, actions: CardActions) {
		this.hoveredId = id;
		this.hoveredActions = actions;
	}

	/**
	 * Clear the hover target if it matches the given ID
	 */
	clearHoverTarget(id: string) {
		if (this.hoveredId === id) {
			this.hoveredId = null;
			this.hoveredActions = null;
		}
	}

	/**
	 * Toggle help overlay
	 */
	toggleHelp() {
		this.helpVisible = !this.helpVisible;
	}

	/**
	 * Get shortcuts for help display
	 */
	getShortcuts(): KeyboardShortcut[] {
		return [
			{ key: '?', description: 'Show/hide keyboard shortcuts' },
			{ key: 'Alt+1', description: 'Add first treatment' },
			{ key: 'Alt+Shift+1', description: 'Remove first treatment' },
			{ key: 'Alt+2', description: 'Add second treatment' },
			{ key: 'Alt+Shift+2', description: 'Remove second treatment' }
		];
	}
}

/**
 * Global keyboard store instance
 */
export const keyboard = new KeyboardStore();
