/**
 * Add-language preference store.
 *
 * Persists the language code that the "+" buttons use when creating new
 * inventory items. Set once per session — useful when batch-entering a
 * single-language binder.
 *
 * Supported codes are the languages Magic: The Gathering is actively printed
 * in, plus Traditional Chinese (zht), which is discontinued but still owned by
 * collectors.
 */

import { browser } from '$app/environment';

export type AddLanguage =
	| 'en'
	| 'es'
	| 'fr'
	| 'de'
	| 'it'
	| 'pt'
	| 'ja'
	| 'ko'
	| 'ru'
	| 'zhs'
	| 'zht';

export interface MtgLanguage {
	code: AddLanguage;
	name: string;
}

/** Display order: English first (default), then alphabetical by English name. */
export const MTG_LANGUAGES: readonly MtgLanguage[] = [
	{ code: 'en', name: 'English' },
	{ code: 'fr', name: 'French' },
	{ code: 'de', name: 'German' },
	{ code: 'it', name: 'Italian' },
	{ code: 'ja', name: 'Japanese' },
	{ code: 'ko', name: 'Korean' },
	{ code: 'pt', name: 'Portuguese' },
	{ code: 'ru', name: 'Russian' },
	{ code: 'zhs', name: 'Simplified Chinese' },
	{ code: 'es', name: 'Spanish' },
	{ code: 'zht', name: 'Traditional Chinese' }
];

const VALID_CODES = new Set<AddLanguage>(MTG_LANGUAGES.map((l) => l.code));

function isAddLanguage(value: string | null): value is AddLanguage {
	return value !== null && VALID_CODES.has(value as AddLanguage);
}

const STORAGE_KEY = 'smc-add-language';

class AddLanguageStore {
	current = $state<AddLanguage>('en');

	constructor() {
		if (browser) {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (isAddLanguage(stored)) {
				this.current = stored;
			}
		}
	}

	set(language: AddLanguage) {
		this.current = language;
		if (browser) {
			localStorage.setItem(STORAGE_KEY, language);
		}
	}
}

export const addLanguage = new AddLanguageStore();
