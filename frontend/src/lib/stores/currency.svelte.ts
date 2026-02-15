/**
 * Currency preference store
 *
 * Manages the user's preferred currency for price display.
 * Persists to localStorage.
 */

import { browser } from '$app/environment';

export type Currency = 'usd' | 'eur';

const STORAGE_KEY = 'priceCurrency';

class CurrencyStore {
	current = $state<Currency>('usd');

	constructor() {
		if (browser) {
			const stored = localStorage.getItem(STORAGE_KEY);
			if (stored === 'usd' || stored === 'eur') {
				this.current = stored;
			}
		}
	}

	set(currency: Currency) {
		this.current = currency;
		if (browser) {
			localStorage.setItem(STORAGE_KEY, currency);
		}
	}

	get symbol(): string {
		return this.current === 'eur' ? '€' : '$';
	}
}

export const currency = new CurrencyStore();
