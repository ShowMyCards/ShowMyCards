import { describe, it, expect } from 'vitest';
import { getCardTreatmentName, getAvailableTreatments } from './card-treatment';

describe('getCardTreatmentName', () => {
	it('should format showcase surge foil correctly', () => {
		const result = getCardTreatmentName(['nonfoil', 'foil'], ['showcase', 'surgefoil'], 'foil');
		expect(result).toBe('Showcase - Surge Foil');
	});

	it('should format nonfoil extended art correctly', () => {
		const result = getCardTreatmentName(['nonfoil'], ['extendedart'], 'nonfoil');
		expect(result).toBe('Extended Art');
	});

	it('should format regular foil without special effects', () => {
		const result = getCardTreatmentName(['nonfoil', 'foil'], [], 'foil');
		expect(result).toBe('Foil');
	});

	it('should format regular nonfoil without special effects', () => {
		const result = getCardTreatmentName(['nonfoil'], [], 'nonfoil');
		expect(result).toBe('Nonfoil');
	});

	it('should handle etched foil', () => {
		const result = getCardTreatmentName(['etched'], [], 'etched');
		expect(result).toBe('Etched');
	});

	it('should format showcase etched foil', () => {
		const result = getCardTreatmentName(['etched'], ['showcase'], 'etched');
		expect(result).toBe('Showcase - Etched');
	});

	it('should handle galaxy foil showcase', () => {
		const result = getCardTreatmentName(['foil'], ['showcase', 'galaxyfoil'], 'foil');
		expect(result).toBe('Showcase - Galaxy Foil');
	});

	it('should handle oil slick foil extended art', () => {
		const result = getCardTreatmentName(['foil'], ['extendedart', 'oilslickfoil'], 'foil');
		expect(result).toBe('Extended Art - Oilslick Foil');
	});

	it('should handle inverted showcase', () => {
		const result = getCardTreatmentName(['nonfoil', 'foil'], ['inverted', 'showcase'], 'nonfoil');
		expect(result).toBe('Inverted - Showcase');
	});

	it('should handle shattered glass foil', () => {
		const result = getCardTreatmentName(['foil'], ['shatteredglass', 'confettifoil'], 'foil');
		expect(result).toBe('Shatteredglass - Confetti Foil');
	});

	it('should only show special foil when foil finish is selected', () => {
		const result = getCardTreatmentName(['nonfoil', 'foil'], ['surgefoil'], 'nonfoil');
		expect(result).toBe('Nonfoil');
	});

	it('should handle multiple special foils (first one wins)', () => {
		const result = getCardTreatmentName(['foil'], ['surgefoil', 'galaxyfoil', 'showcase'], 'foil');
		expect(result).toBe('Showcase - Surge Foil');
	});

	it('should handle glossy finish', () => {
		const result = getCardTreatmentName(['glossy'], [], 'glossy');
		expect(result).toBe('Glossy');
	});

	it('should handle halo foil', () => {
		const result = getCardTreatmentName(['foil'], ['halofoil'], 'foil');
		expect(result).toBe('Halo Foil');
	});

	it('should handle raised foil showcase', () => {
		const result = getCardTreatmentName(['foil'], ['showcase', 'raisedfoil'], 'foil');
		expect(result).toBe('Showcase - Raised Foil');
	});

	it('should handle all special foil variants', () => {
		const foilVariants = [
			{ effect: 'surgefoil', expected: 'Surge Foil' },
			{ effect: 'galaxyfoil', expected: 'Galaxy Foil' },
			{ effect: 'oilslickfoil', expected: 'Oilslick Foil' },
			{ effect: 'confettifoil', expected: 'Confetti Foil' },
			{ effect: 'halofoil', expected: 'Halo Foil' },
			{ effect: 'raisedfoil', expected: 'Raised Foil' },
			{ effect: 'ripplefoil', expected: 'Ripple Foil' },
			{ effect: 'fracturefoil', expected: 'Fracture Foil' },
			{ effect: 'manafoil', expected: 'Mana Foil' },
			{ effect: 'firstplacefoil', expected: 'Firstplace Foil' },
			{ effect: 'dragonscalefoil', expected: 'Dragonscale Foil' },
			{ effect: 'singularityfoil', expected: 'Singularity Foil' },
			{ effect: 'cosmicfoil', expected: 'Cosmic Foil' },
			{ effect: 'chocobofoil', expected: 'Chocobo Foil' }
		];

		foilVariants.forEach(({ effect, expected }) => {
			const result = getCardTreatmentName(['foil'], [effect], 'foil');
			expect(result).toBe(expected);
		});
	});

	it('should handle all style effects', () => {
		const styleEffects = [
			{ effect: 'inverted', expected: 'Inverted' },
			{ effect: 'showcase', expected: 'Showcase' },
			{ effect: 'extendedart', expected: 'Extended Art' },
			{ effect: 'shatteredglass', expected: 'Shatteredglass' }
		];

		styleEffects.forEach(({ effect, expected }) => {
			const result = getCardTreatmentName(['nonfoil'], [effect], 'nonfoil');
			expect(result).toBe(expected);
		});
	});
});

describe('getAvailableTreatments', () => {
	it('should return all available treatments with correct names', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], ['showcase', 'surgefoil']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Showcase' },
			{ key: 'foil', name: 'Showcase - Surge Foil' }
		]);
	});

	it('should handle single finish with extended art', () => {
		const result = getAvailableTreatments(['nonfoil'], ['extendedart']);
		expect(result).toEqual([{ key: 'nonfoil', name: 'Extended Art' }]);
	});

	it('should handle cards with no special effects', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], []);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Nonfoil' },
			{ key: 'foil', name: 'Foil' }
		]);
	});

	it('should handle etched foil with showcase', () => {
		const result = getAvailableTreatments(['etched'], ['showcase']);
		expect(result).toEqual([{ key: 'etched', name: 'Showcase - Etched' }]);
	});

	it('should handle multiple finishes with galaxy foil', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], ['galaxyfoil', 'extendedart']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Extended Art' },
			{ key: 'foil', name: 'Extended Art - Galaxy Foil' }
		]);
	});
});
