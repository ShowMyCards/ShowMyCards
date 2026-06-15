import { describe, it, expect } from 'vitest';
import { convertManaboxLine, preprocessImportText } from './manabox';

describe('convertManaboxLine', () => {
	it('converts a non-foil ManaBox line to Scryfall syntax', () => {
		expect(convertManaboxLine('1 Lightning Bolt (2ED) 161')).toBe('1 e:2ED cn:161');
	});

	it('converts a foil ManaBox line and adds the foil marker', () => {
		expect(convertManaboxLine('2 Counterspell (EMA) 43 *F*')).toBe('2! e:EMA cn:43');
	});

	it('handles card names containing parentheses by using the first group', () => {
		// The set code is always the parenthesised group immediately before the
		// collector number in ManaBox exports.
		expect(convertManaboxLine('3 Fire // Ice (APC) 128')).toBe('3 e:APC cn:128');
	});

	it('returns null for a Scryfall-style line (no parenthesised set code)', () => {
		expect(convertManaboxLine('4 e:who cn:1056')).toBeNull();
		expect(convertManaboxLine('2! !"Lightning Bolt"')).toBeNull();
		expect(convertManaboxLine('sol ring')).toBeNull();
	});
});

describe('preprocessImportText', () => {
	it('appends the language clause to converted ManaBox lines', () => {
		expect(preprocessImportText('1 Lightning Bolt (2ED) 161', 'de')).toBe('1 e:2ED cn:161 l:de');
	});

	it('appends the language clause to passed-through Scryfall lines', () => {
		expect(preprocessImportText('4 e:who cn:1056', 'en')).toBe('4 e:who cn:1056 l:en');
	});

	it('does not double-append when a language clause is already present', () => {
		expect(preprocessImportText('4 e:who cn:1056 l:ja', 'en')).toBe('4 e:who cn:1056 l:ja');
		expect(preprocessImportText('4 e:who lang:fr', 'en')).toBe('4 e:who lang:fr');
	});

	it('leaves comment and blank lines untouched', () => {
		const input = '# a comment\n\n// another comment';
		expect(preprocessImportText(input, 'en')).toBe(input);
	});

	it('does not append a language clause when none is given', () => {
		expect(preprocessImportText('1 Lightning Bolt (2ED) 161', '')).toBe('1 e:2ED cn:161');
	});

	it('converts only the ManaBox lines in mixed input', () => {
		const input = [
			'1 Lightning Bolt (2ED) 161',
			'4 e:who cn:1056',
			'2 Counterspell (EMA) 43 *F*'
		].join('\n');
		expect(preprocessImportText(input, 'en')).toBe(
			['1 e:2ED cn:161 l:en', '4 e:who cn:1056 l:en', '2! e:EMA cn:43 l:en'].join('\n')
		);
	});
});
