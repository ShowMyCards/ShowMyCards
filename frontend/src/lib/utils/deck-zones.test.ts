import { describe, expect, it } from 'vitest';
import { PICKABLE_ZONES, ZONE_LABELS, zonesLabel } from './deck-zones';

describe('zonesLabel', () => {
	it('labels a single zone', () => {
		expect(zonesLabel(['side'])).toBe('Side');
	});

	it('labels Other and joins several zones in order', () => {
		expect(zonesLabel(['main', 'other'])).toBe('Main, Other');
	});

	it('returns an empty string for no zones', () => {
		expect(zonesLabel([])).toBe('');
	});
});

describe('PICKABLE_ZONES', () => {
	it('offers every zone except Other', () => {
		expect([...PICKABLE_ZONES].sort()).toEqual(
			Object.keys(ZONE_LABELS)
				.filter((z) => z !== 'other')
				.sort()
		);
	});
});
