import { describe, expect, it } from 'vitest';
import { manaSymbolClasses } from './mana';

describe('manaSymbolClasses', () => {
	it('returns an empty array for empty/undefined input', () => {
		expect(manaSymbolClasses('')).toEqual([]);
		expect(manaSymbolClasses(undefined)).toEqual([]);
		expect(manaSymbolClasses(null)).toEqual([]);
	});

	it('maps generic and coloured symbols in order', () => {
		expect(manaSymbolClasses('{2}{G}{G}')).toEqual([
			'ms ms-2 ms-cost',
			'ms ms-g ms-cost',
			'ms ms-g ms-cost'
		]);
	});

	it('maps X and hybrid symbols', () => {
		expect(manaSymbolClasses('{X}{U/R}')).toEqual(['ms ms-x ms-cost', 'ms ms-ur ms-cost']);
	});

	it('maps phyrexian and twobrid symbols', () => {
		expect(manaSymbolClasses('{G/P}')).toEqual(['ms ms-gp ms-cost']);
		expect(manaSymbolClasses('{2/W}')).toEqual(['ms ms-2w ms-cost']);
	});

	it('maps the special half and infinity glyphs', () => {
		expect(manaSymbolClasses('{½}')).toEqual(['ms ms-half ms-cost']);
		expect(manaSymbolClasses('{∞}')).toEqual(['ms ms-infinity ms-cost']);
	});
});
