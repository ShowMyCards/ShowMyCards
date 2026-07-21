import { describe, expect, it } from 'vitest';
import { parseCardList, type ParsedCard } from './card-list-parser';

// Real decklist exports collected as fixtures (see FR98 issue 3). Imported raw so
// the parser is exercised against genuine formats, not synthesised approximations.
import moxfieldAnyPrintings from './__fixtures__/decklists/moxfield-any-printings-mtgo.txt?raw';
import moxfieldSpecificPrintings from './__fixtures__/decklists/moxfield-specific-printings.txt?raw';
import moxfieldMtga from './__fixtures__/decklists/moxfield-mtga.txt?raw';
import mtggoldfish from './__fixtures__/decklists/mtggoldfish-any-printings.txt?raw';

function byName(cards: ParsedCard[], name: string): ParsedCard | undefined {
	return cards.find((c) => c.name === name);
}

describe('parseCardList — Scryfall syntax (back-compat)', () => {
	it('parses quantity and leading treatment markers', () => {
		const { cards } = parseCardList('4 e:who cn:1056\n2! e:who cn:1056\n1!! e:cmr cn:361');
		expect(cards).toHaveLength(3);
		expect(cards[0]).toMatchObject({ quantity: 4, treatment: 'nonfoil', query: 'e:who cn:1056' });
		expect(cards[1]).toMatchObject({ quantity: 2, treatment: 'foil' });
		expect(cards[2]).toMatchObject({ quantity: 1, treatment: 'etched' });
	});

	it('defaults quantity to 1 and preserves a space-separated exact-name query', () => {
		// The leading foil marker and the exact-match query are space-separated, per
		// the import page's documented syntax (e.g. "2! !\"Lightning Bolt\"").
		const { cards } = parseCardList('sol ring\n2! !"Lightning Bolt"');
		expect(cards[0]).toMatchObject({ quantity: 1, query: 'sol ring', pinned: false });
		expect(cards[1]).toMatchObject({ quantity: 2, treatment: 'foil', query: '!"Lightning Bolt"' });
		// A query with Scryfall operators is not exposed as a display name.
		expect(cards[1].name).toBeUndefined();
	});

	it('does not misparse Scryfall queries containing parentheses as pinned printings', () => {
		const { cards } = parseCardList('4 (o:draw or o:scry) e:dom');
		expect(cards[0].pinned).toBe(false);
		expect(cards[0].query).toBe('(o:draw or o:scry) e:dom');
	});

	it('binds the set to the LAST parenthesised group for names containing parentheses', () => {
		// Preserves the ManaBox shim guarantee now that parsing is native.
		const { cards } = parseCardList('1 Erase (Not the One) (WTH) 20');
		expect(cards[0]).toMatchObject({
			name: 'Erase (Not the One)',
			set: 'WTH',
			collectorNumber: '20',
			pinned: true
		});
	});
});

describe('parseCardList — decklist identity', () => {
	it('parses a pinned printing with a foil marker', () => {
		const { cards } = parseCardList("1 Ashnod's Altar (BRR) 67 *F*");
		expect(cards[0]).toMatchObject({
			quantity: 1,
			name: "Ashnod's Altar",
			set: 'BRR',
			collectorNumber: '67',
			treatment: 'foil',
			pinned: true,
			query: 'e:BRR cn:67'
		});
	});

	it('parses an etched marker', () => {
		const { cards } = parseCardList('1 Gravecrawler (2X2) 438 *E*');
		expect(cards[0]).toMatchObject({ treatment: 'etched', pinned: true, query: 'e:2X2 cn:438' });
	});

	it('handles collector numbers with letter suffixes', () => {
		const { cards } = parseCardList(
			'1 Sothera, the Supervoid (PEOE) 115s *F*\n1 The Meathook Massacre (PMID) 112p *F*'
		);
		expect(cards[0].collectorNumber).toBe('115s');
		expect(cards[0].query).toBe('e:PEOE cn:115s');
		expect(cards[1].collectorNumber).toBe('112p');
	});

	it('keeps a double-faced name intact and pins by set/collector', () => {
		const { cards } = parseCardList(
			"1 Agadeem's Awakening / Agadeem, the Undercrypt (ZNR) 336 *F*"
		);
		expect(cards[0].name).toBe("Agadeem's Awakening / Agadeem, the Undercrypt");
		expect(cards[0]).toMatchObject({ set: 'ZNR', collectorNumber: '336', pinned: true });
	});

	it('treats a bare name as any-printing (not pinned)', () => {
		const { cards } = parseCardList('1 Afterlife Insurance');
		expect(cards[0]).toMatchObject({ name: 'Afterlife Insurance', pinned: false, quantity: 1 });
		expect(cards[0].set).toBeUndefined();
	});

	it('treats a slash split-card name as a plain name', () => {
		const { cards } = parseCardList('2 Wear/Tear');
		expect(cards[0]).toMatchObject({ name: 'Wear/Tear', pinned: false, quantity: 2 });
	});

	it('appends the language clause to resolved queries', () => {
		const { cards } = parseCardList("1 Ashnod's Altar (BRR) 67\nsol ring", { language: 'de' });
		expect(cards[0].query).toBe('e:BRR cn:67 l:de');
		expect(cards[1].query).toBe('sol ring l:de');
	});

	it('does not double-append a language clause', () => {
		const { cards } = parseCardList('sol ring l:en', { language: 'en' });
		expect(cards[0].query).toBe('sol ring l:en');
	});
});

describe('parseCardList — zones from headers', () => {
	it('switches zones on section headers and skips the About metadata block', () => {
		const input = [
			'About',
			'Name Triggers!',
			'',
			'Commander',
			'1 Teysa Karlov',
			'',
			'Deck',
			'1 Sol Ring',
			'',
			'Sideboard',
			'1 Damn',
			'',
			'Maybeboard',
			'1 Toxic Deluge'
		].join('\n');
		const { cards } = parseCardList(input);
		// The About block's "Name Triggers!" line must not become a card.
		expect(cards.find((c) => c.name === 'Name Triggers!')).toBeUndefined();
		expect(byName(cards, 'Teysa Karlov')?.zone).toBe('command');
		expect(byName(cards, 'Sol Ring')?.zone).toBe('main');
		expect(byName(cards, 'Damn')?.zone).toBe('side');
		expect(byName(cards, 'Toxic Deluge')?.zone).toBe('maybe');
	});

	it('accepts header variants with trailing counts', () => {
		const input = 'Commander {1}\n1 Teysa Karlov\n\nSideboard (2)\n1 Damn';
		const { cards } = parseCardList(input);
		expect(byName(cards, 'Teysa Karlov')?.zone).toBe('command');
		expect(byName(cards, 'Damn')?.zone).toBe('side');
	});

	it('skips non-zone category headers (e.g. Archidekt "Creatures {30}") without emitting cards', () => {
		const input = 'Deck\nCreatures {30}\n1 Blood Artist\nLands {38}\n1 Command Tower';
		const { cards } = parseCardList(input);
		expect(cards).toHaveLength(2);
		expect(byName(cards, 'Blood Artist')?.zone).toBe('main');
		expect(byName(cards, 'Command Tower')?.zone).toBe('main');
	});
});

describe('parseCardList — zones from blank-line blocks (headerless)', () => {
	it('assigns the first block to Main and later blocks to Side', () => {
		const input = '1 Lightning Bolt\n1 Ragavan\n\n1 Wear/Tear\n1 High Noon';
		const { cards } = parseCardList(input);
		expect(byName(cards, 'Lightning Bolt')?.zone).toBe('main');
		expect(byName(cards, 'Ragavan')?.zone).toBe('main');
		expect(byName(cards, 'Wear/Tear')?.zone).toBe('side');
		expect(byName(cards, 'High Noon')?.zone).toBe('side');
	});

	it('lets the same card appear in both Main and Side (split across zones)', () => {
		const input = '1 The Legend of Roku\n\n1 The Legend of Roku';
		const { cards } = parseCardList(input);
		const zones = cards.filter((c) => c.name === 'The Legend of Roku').map((c) => c.zone);
		expect(zones).toEqual(['main', 'side']);
	});
});

describe('parseCardList — real export fixtures', () => {
	it('Moxfield "Any Printings" (MTGO): all any-printing, commander block → side', () => {
		const { cards, errors } = parseCardList(moxfieldAnyPrintings);
		expect(errors).toHaveLength(0);
		expect(cards.every((c) => !c.pinned)).toBe(true);
		// The trailing single-card block is the commander in MTGO's sideboard slot.
		const teysa = byName(cards, 'Teysa Karlov');
		expect(teysa?.zone).toBe('side');
		// Basic lands keep their quantities.
		expect(byName(cards, 'Swamp')?.quantity).toBe(7);
		expect(byName(cards, 'Plains')?.quantity).toBe(4);
	});

	it('Moxfield "Specific Printings": every line pinned with a set/collector', () => {
		const { cards, errors } = parseCardList(moxfieldSpecificPrintings);
		expect(errors).toHaveLength(0);
		expect(cards.every((c) => c.pinned && !!c.set && !!c.collectorNumber)).toBe(true);
		expect(byName(cards, "Ashnod's Altar")).toMatchObject({ set: 'BRR', treatment: 'foil' });
		// Etched lines resolve as etched.
		expect(byName(cards, 'Toxic Deluge')?.treatment).toBe('etched');
	});

	it('Moxfield MTGA: header-driven zones with the About block skipped', () => {
		const { cards, errors } = parseCardList(moxfieldMtga);
		expect(errors).toHaveLength(0);
		expect(cards.find((c) => c.name === 'Name Triggers!')).toBeUndefined();
		expect(byName(cards, 'Teysa Karlov')?.zone).toBe('command');
		expect(byName(cards, "Ashnod's Altar")?.zone).toBe('main');
		expect(cards.every((c) => !c.pinned)).toBe(true);
	});

	it('MTGGoldfish: mainboard block → main, sideboard block → side', () => {
		const { cards, errors } = parseCardList(mtggoldfish);
		expect(errors).toHaveLength(0);
		expect(byName(cards, 'Ragavan, Nimble Pilferer')?.zone).toBe('main');
		expect(byName(cards, 'Celestial Purge')?.zone).toBe('side');
		// "The Legend of Roku" is in both the mainboard and the sideboard.
		const roku = cards.filter((c) => c.name === 'The Legend of Roku').map((c) => c.zone);
		expect(roku).toEqual(['main', 'side']);
	});
});
