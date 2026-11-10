import { describe, it, expect } from 'vitest';
import { scryfallImageUrl } from './scryfall-image';

describe('scryfallImageUrl', () => {
	const id = '937dbc51-b589-4237-9fce-ea5c757f7c48';

	it('defaults to the grid WebP version', () => {
		expect(scryfallImageUrl(id)).toBe(
			`https://api.scryfall.com/cards/${id}?format=image&version=grid`
		);
	});

	it('uses the requested version', () => {
		expect(scryfallImageUrl(id, 'thumb')).toBe(
			`https://api.scryfall.com/cards/${id}?format=image&version=thumb`
		);
	});
});
