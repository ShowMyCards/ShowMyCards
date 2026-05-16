import { describe, it, expect } from 'vitest';
import { buildSearchQuery } from './search-builder';
import type { SearchBuilderParams } from './search-builder';

function params(overrides: Partial<SearchBuilderParams> = {}): SearchBuilderParams {
	return {
		cardName: '',
		setCode: '',
		cardType: '',
		colors: new Set(),
		colorMode: 'all',
		numericValues: { mv: '', pow: '', loy: '', tou: '' },
		numericOps: { mv: '=', pow: '=', loy: '=', tou: '=' },
		...overrides
	};
}

describe('buildSearchQuery', () => {
	describe('card name', () => {
		it('single word uses bare name: syntax', () => {
			expect(buildSearchQuery(params({ cardName: 'Lightning' }))).toBe('name:Lightning');
		});

		it('multiple words are quoted', () => {
			expect(buildSearchQuery(params({ cardName: 'Lightning Bolt' }))).toBe(
				'name:"Lightning Bolt"'
			);
		});

		it('trims surrounding whitespace', () => {
			expect(buildSearchQuery(params({ cardName: '  Opt  ' }))).toBe('name:Opt');
		});

		it('whitespace-only is omitted', () => {
			expect(buildSearchQuery(params({ cardName: '   ' }))).toBe('');
		});
	});

	describe('set', () => {
		it('set code is included as s: term', () => {
			expect(buildSearchQuery(params({ setCode: 'blb' }))).toBe('s:blb');
		});

		it('trims surrounding whitespace', () => {
			expect(buildSearchQuery(params({ setCode: ' m21 ' }))).toBe('s:m21');
		});

		it('empty set code is omitted', () => {
			expect(buildSearchQuery(params({ setCode: '' }))).toBe('');
		});
	});

	describe('card type', () => {
		it('single word uses bare t: syntax', () => {
			expect(buildSearchQuery(params({ cardType: 'Creature' }))).toBe('t:Creature');
		});

		it('multiple words are quoted', () => {
			expect(buildSearchQuery(params({ cardType: 'Legendary Creature' }))).toBe(
				't:"Legendary Creature"'
			);
		});

		it('trims surrounding whitespace', () => {
			expect(buildSearchQuery(params({ cardType: '  Instant  ' }))).toBe('t:Instant');
		});
	});

	describe('colors — mode "all"', () => {
		it('single color', () => {
			expect(buildSearchQuery(params({ colors: new Set(['u']), colorMode: 'all' }))).toBe('c:u');
		});

		it('multiple colors follow WUBRG order', () => {
			expect(buildSearchQuery(params({ colors: new Set(['r', 'w', 'u']), colorMode: 'all' }))).toBe(
				'c:wur'
			);
		});

		it('colorless', () => {
			expect(buildSearchQuery(params({ colors: new Set(['c']), colorMode: 'all' }))).toBe('c:c');
		});

		it('all five colors', () => {
			expect(
				buildSearchQuery(params({ colors: new Set(['g', 'b', 'r', 'u', 'w']), colorMode: 'all' }))
			).toBe('c:wubrg');
		});
	});

	describe('colors — mode "exactly"', () => {
		it('single color uses = operator', () => {
			expect(buildSearchQuery(params({ colors: new Set(['g']), colorMode: 'exactly' }))).toBe(
				'c=g'
			);
		});

		it('multiple colors use = operator in order', () => {
			expect(buildSearchQuery(params({ colors: new Set(['g', 'w']), colorMode: 'exactly' }))).toBe(
				'c=wg'
			);
		});
	});

	describe('colors — mode "any"', () => {
		it('single color uses bare c: syntax', () => {
			expect(buildSearchQuery(params({ colors: new Set(['r']), colorMode: 'any' }))).toBe('c:r');
		});

		it('multiple colors use OR syntax', () => {
			expect(buildSearchQuery(params({ colors: new Set(['r', 'g']), colorMode: 'any' }))).toBe(
				'(c:r or c:g)'
			);
		});

		it('three colors use OR syntax in WUBRG order', () => {
			expect(buildSearchQuery(params({ colors: new Set(['g', 'u', 'r']), colorMode: 'any' }))).toBe(
				'(c:u or c:r or c:g)'
			);
		});
	});

	describe('numeric fields', () => {
		it('mana value with equals', () => {
			expect(
				buildSearchQuery(params({ numericValues: { mv: '3', pow: '', loy: '', tou: '' } }))
			).toBe('cmc=3');
		});

		it('power with greater-than', () => {
			expect(
				buildSearchQuery(
					params({
						numericValues: { mv: '', pow: '2', loy: '', tou: '' },
						numericOps: { mv: '=', pow: '>', loy: '=', tou: '=' }
					})
				)
			).toBe('pow>2');
		});

		it('toughness with less-than', () => {
			expect(
				buildSearchQuery(
					params({
						numericValues: { mv: '', pow: '', loy: '', tou: '5' },
						numericOps: { mv: '=', pow: '=', loy: '=', tou: '<' }
					})
				)
			).toBe('tou<5');
		});

		it('loyalty with equals', () => {
			expect(
				buildSearchQuery(params({ numericValues: { mv: '', pow: '', loy: '4', tou: '' } }))
			).toBe('loy=4');
		});

		it('empty numeric value is omitted', () => {
			expect(
				buildSearchQuery(params({ numericValues: { mv: '', pow: '', loy: '', tou: '' } }))
			).toBe('');
		});
	});

	describe('combinations', () => {
		it('name + set + single color', () => {
			expect(
				buildSearchQuery(
					params({ cardName: 'Opt', setCode: 'blb', colors: new Set(['u']), colorMode: 'all' })
				)
			).toBe('name:Opt s:blb c:u');
		});

		it('all fields produce a correctly joined query', () => {
			expect(
				buildSearchQuery({
					cardName: 'Lightning Bolt',
					setCode: 'm10',
					cardType: 'Instant',
					colors: new Set(['r']),
					colorMode: 'exactly',
					numericValues: { mv: '1', pow: '', loy: '', tou: '' },
					numericOps: { mv: '=', pow: '=', loy: '=', tou: '=' }
				})
			).toBe('name:"Lightning Bolt" s:m10 t:Instant c=r cmc=1');
		});
	});

	describe('edge cases', () => {
		it('all fields empty returns empty string', () => {
			expect(buildSearchQuery(params())).toBe('');
		});

		it('empty colors set is omitted regardless of color mode', () => {
			expect(buildSearchQuery(params({ colors: new Set(), colorMode: 'exactly' }))).toBe('');
		});
	});
});
