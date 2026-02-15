/**
 * Audio manager for application sounds
 *
 * Preloads audio files and provides methods to play them on demand.
 * Respects user settings for sound preferences.
 *
 * @example
 * ```ts
 * import { audio } from '$lib';
 *
 * // Play a sound
 * audio.play('match');
 *
 * // Enable/disable sounds
 * audio.setEnabled(false);
 * ```
 */

type SoundName = 'match' | 'success' | 'error';

const SOUND_PATHS: Record<SoundName, string> = {
	match: '/sounds/match.ogg',
	success: '/sounds/success.ogg',
	error: '/sounds/error.ogg'
};

class AudioManager {
	/** Preloaded audio elements */
	private sounds: Map<SoundName, HTMLAudioElement> = new Map();

	/** Whether sounds are enabled */
	private enabled = true;

	/** Volume level (0-1) */
	private volume = 0.5;

	/**
	 * Initialize audio manager by preloading sounds
	 * Call this once when the app starts
	 */
	init() {
		if (typeof window === 'undefined') return;

		for (const [name, path] of Object.entries(SOUND_PATHS)) {
			const audio = new Audio(path);
			audio.preload = 'auto';
			audio.volume = this.volume;
			this.sounds.set(name as SoundName, audio);
		}
	}

	/**
	 * Play a sound by name
	 */
	play(name: SoundName) {
		if (!this.enabled) return;

		const sound = this.sounds.get(name);
		if (!sound) return;

		// Clone the audio element to allow overlapping sounds
		const clone = sound.cloneNode() as HTMLAudioElement;
		clone.volume = this.volume;
		clone.play().catch(() => {
			// Ignore play errors (e.g., autoplay restrictions)
		});
	}

	/**
	 * Set whether sounds are enabled
	 */
	setEnabled(enabled: boolean) {
		this.enabled = enabled;
	}

	/**
	 * Check if sounds are enabled
	 */
	isEnabled(): boolean {
		return this.enabled;
	}

	/**
	 * Set volume level (0-1)
	 */
	setVolume(volume: number) {
		this.volume = Math.max(0, Math.min(1, volume));
		for (const sound of this.sounds.values()) {
			sound.volume = this.volume;
		}
	}

	/**
	 * Get current volume level
	 */
	getVolume(): number {
		return this.volume;
	}
}

/**
 * Global audio manager instance
 */
export const audio = new AudioManager();
