import { describe, it, expect } from 'vitest';
import { convertManaboxLine, preprocessImportText } from './manabox';

describe('convertManaboxLine', () => {
	it('converts a non-foil ManaBox line to Scryfall syntax', () => {
		expect(convertManaboxLine('1 Lightning Bolt (2ED) 161')).toBe('1 e:2ED cn:161');
	});

	it('converts a foil ManaBox line and adds the foil marker', () => {
		expect(convertManaboxLine('2 Counterspell (EMA) 43 *F*')).toBe('2! e:EMA cn:43');
	});

	it('converts an etched ManaBox line and adds the etched marker', () => {
		expect(convertManaboxLine('1 Sol Ring (LTC) 284 *E*')).toBe('1!! e:LTC cn:284');
	});

	it('handles split-card names with no parentheses', () => {
		expect(convertManaboxLine('3 Fire // Ice (APC) 128')).toBe('3 e:APC cn:128');
	});

	it('binds the set code to the last group when the name contains parentheses', () => {
		// The set code is always the parenthesised group immediately before the
		// collector number, so a name that itself contains parentheses must not
		// be mistaken for the set code.
		expect(convertManaboxLine("1 Erase (Not the Urza's One) (WTH) 20")).toBe('1 e:WTH cn:20');
	});

	it('handles star (foil-sheet) collector numbers', () => {
		expect(convertManaboxLine('1 Arcane Signet (WOC) 118★')).toBe('1 e:WOC cn:118★');
	});

	it('returns null for a Scryfall-style line (no parenthesised set code)', () => {
		expect(convertManaboxLine('4 e:who cn:1056')).toBeNull();
		expect(convertManaboxLine('2! !"Lightning Bolt"')).toBeNull();
		expect(convertManaboxLine('sol ring')).toBeNull();
	});

	it('does not swallow Scryfall queries that contain parentheses', () => {
		// Regression guard: valid Scryfall grouping syntax must pass through
		// untouched, not be mis-parsed as a ManaBox line.
		expect(convertManaboxLine('1 (t:creature) e:dom')).toBeNull();
		expect(convertManaboxLine('4 (o:draw or o:scry) e:dom')).toBeNull();
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

	it('leaves parenthesised Scryfall queries intact while appending language', () => {
		expect(preprocessImportText('1 (t:creature) e:dom', 'en')).toBe('1 (t:creature) e:dom l:en');
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
