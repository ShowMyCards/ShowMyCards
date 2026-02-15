import { describe, it, expect } from 'vitest';
import { getCardTreatmentName, getAvailableTreatments } from './card-treatment';

describe('getCardTreatmentName', () => {
	describe('base finishes without frame effects', () => {
		it('should return "Nonfoil" for nonfoil cards with no effects', () => {
			const result = getCardTreatmentName(['nonfoil'], [], 'nonfoil');
			expect(result).toBe('Nonfoil');
		});

		it('should return "Foil" for foil cards with no effects', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], [], 'foil');
			expect(result).toBe('Foil');
		});

		it('should return "Etched" for etched cards with no effects', () => {
			const result = getCardTreatmentName(['etched'], [], 'etched');
			expect(result).toBe('Etched');
		});

		it('should return "Glossy" for glossy cards with no effects', () => {
			const result = getCardTreatmentName(['glossy'], [], 'glossy');
			expect(result).toBe('Glossy');
		});

		it('should default to nonfoil when finish not specified', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil']);
			expect(result).toBe('Nonfoil');
		});
	});

	describe('special foil types', () => {
		it('should handle surgefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['surgefoil'], 'foil');
			expect(result).toBe('Surge Foil');
		});

		it('should handle galaxyfoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['galaxyfoil'], 'foil');
			expect(result).toBe('Galaxy Foil');
		});

		it('should handle oilslickfoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['oilslickfoil'], 'foil');
			expect(result).toBe('Oilslick Foil');
		});

		it('should handle confettifoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['confettifoil'], 'foil');
			expect(result).toBe('Confetti Foil');
		});

		it('should handle halofoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['halofoil'], 'foil');
			expect(result).toBe('Halo Foil');
		});

		it('should handle raisedfoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['raisedfoil'], 'foil');
			expect(result).toBe('Raised Foil');
		});

		it('should handle ripplefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['ripplefoil'], 'foil');
			expect(result).toBe('Ripple Foil');
		});

		it('should handle fracturefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['fracturefoil'], 'foil');
			expect(result).toBe('Fracture Foil');
		});

		it('should handle manafoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['manafoil'], 'foil');
			expect(result).toBe('Mana Foil');
		});

		it('should handle firstplacefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['firstplacefoil'], 'foil');
			expect(result).toBe('Firstplace Foil');
		});

		it('should handle dragonscalefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['dragonscalefoil'], 'foil');
			expect(result).toBe('Dragonscale Foil');
		});

		it('should handle singularityfoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['singularityfoil'], 'foil');
			expect(result).toBe('Singularity Foil');
		});

		it('should handle cosmicfoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['cosmicfoil'], 'foil');
			expect(result).toBe('Cosmic Foil');
		});

		it('should handle chocobofoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['chocobofoil'], 'foil');
			expect(result).toBe('Chocobo Foil');
		});

		it('should not show special foils for nonfoil finish', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['surgefoil'], 'nonfoil');
			expect(result).toBe('Nonfoil');
		});
	});

	describe('style effects', () => {
		it('should handle showcase', () => {
			const result = getCardTreatmentName(['nonfoil'], ['showcase'], 'nonfoil');
			expect(result).toBe('Showcase');
		});

		it('should handle extendedart', () => {
			const result = getCardTreatmentName(['nonfoil'], ['extendedart'], 'nonfoil');
			expect(result).toBe('Extended Art');
		});

		it('should handle inverted', () => {
			const result = getCardTreatmentName(['nonfoil'], ['inverted'], 'nonfoil');
			expect(result).toBe('Inverted');
		});

		it('should handle shatteredglass', () => {
			const result = getCardTreatmentName(['nonfoil'], ['shatteredglass'], 'nonfoil');
			expect(result).toBe('Shatteredglass');
		});

		it('should handle showcase with foil finish', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['showcase'], 'foil');
			expect(result).toBe('Showcase - Foil');
		});

		it('should handle extendedart with foil finish', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['extendedart'], 'foil');
			expect(result).toBe('Extended Art - Foil');
		});
	});

	describe('combination scenarios', () => {
		it('should handle showcase + surgefoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['showcase', 'surgefoil'], 'foil');
			expect(result).toBe('Showcase - Surge Foil');
		});

		it('should handle showcase + surgefoil with nonfoil finish', () => {
			const result = getCardTreatmentName(
				['nonfoil', 'foil'],
				['showcase', 'surgefoil'],
				'nonfoil'
			);
			expect(result).toBe('Showcase');
		});

		it('should handle extendedart + galaxyfoil', () => {
			const result = getCardTreatmentName(
				['nonfoil', 'foil'],
				['extendedart', 'galaxyfoil'],
				'foil'
			);
			expect(result).toBe('Extended Art - Galaxy Foil');
		});

		it('should handle multiple style effects', () => {
			const result = getCardTreatmentName(
				['nonfoil', 'foil'],
				['showcase', 'extendedart', 'surgefoil'],
				'foil'
			);
			// Should show both style effects and the special foil
			expect(result).toBe('Showcase - Extended Art - Surge Foil');
		});

		it('should handle inverted + halofoil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['inverted', 'halofoil'], 'foil');
			expect(result).toBe('Inverted - Halo Foil');
		});
	});

	describe('edge cases', () => {
		it('should handle empty frame effects array', () => {
			const result = getCardTreatmentName(['nonfoil'], [], 'nonfoil');
			expect(result).toBe('Nonfoil');
		});

		it('should handle undefined frame effects', () => {
			const result = getCardTreatmentName(['nonfoil'], undefined, 'nonfoil');
			expect(result).toBe('Nonfoil');
		});

		it('should handle unknown frame effects', () => {
			const result = getCardTreatmentName(['nonfoil'], ['unknown', 'weird'], 'nonfoil');
			expect(result).toBe('Nonfoil');
		});

		it('should handle empty finishes array', () => {
			const result = getCardTreatmentName([], ['showcase'], 'nonfoil');
			expect(result).toBe('Showcase');
		});

		it('should handle case variations in frame effects', () => {
			// Note: The function finds effects case-insensitively but preserves
			// the original case in the first character of each word after splitting
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['surgefoil', 'showcase'], 'foil');
			expect(result).toBe('Showcase - Surge Foil');
		});

		it('should handle etched with special foil', () => {
			const result = getCardTreatmentName(['etched'], ['surgefoil'], 'etched');
			expect(result).toBe('Surge Foil');
		});

		it('should handle glossy with special foil', () => {
			const result = getCardTreatmentName(['glossy'], ['galaxyfoil'], 'glossy');
			expect(result).toBe('Galaxy Foil');
		});

		it('should handle multiple special foils (only use first one found)', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['surgefoil', 'galaxyfoil'], 'foil');
			// Should use the first special foil found
			expect(result).toBe('Surge Foil');
		});
	});

	describe('real-world examples', () => {
		it('should handle standard foil Collector Booster card', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['extendedart'], 'foil');
			expect(result).toBe('Extended Art - Foil');
		});

		it('should handle Brothers War showcase card', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['showcase', 'surgefoil'], 'foil');
			expect(result).toBe('Showcase - Surge Foil');
		});

		it('should handle March of the Machine halo foil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['halofoil'], 'foil');
			expect(result).toBe('Halo Foil');
		});

		it('should handle Phyrexia All Will Be One oil slick foil', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['oilslickfoil'], 'foil');
			expect(result).toBe('Oilslick Foil');
		});

		it('should handle standard nonfoil showcase', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], ['showcase'], 'nonfoil');
			expect(result).toBe('Showcase');
		});

		it('should handle regular card with no special treatments', () => {
			const result = getCardTreatmentName(['nonfoil', 'foil'], [], 'nonfoil');
			expect(result).toBe('Nonfoil');
		});
	});
});

describe('getAvailableTreatments', () => {
	it('should return treatments for single finish', () => {
		const result = getAvailableTreatments(['nonfoil'], []);
		expect(result).toEqual([{ key: 'nonfoil', name: 'Nonfoil' }]);
	});

	it('should return treatments for multiple finishes', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], []);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Nonfoil' },
			{ key: 'foil', name: 'Foil' }
		]);
	});

	it('should apply frame effects to each finish', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], ['showcase']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Showcase' },
			{ key: 'foil', name: 'Showcase - Foil' }
		]);
	});

	it('should handle special foils for foil finish', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], ['showcase', 'surgefoil']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Showcase' },
			{ key: 'foil', name: 'Showcase - Surge Foil' }
		]);
	});

	it('should handle etched and glossy finishes', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil', 'etched', 'glossy'], []);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Nonfoil' },
			{ key: 'foil', name: 'Foil' },
			{ key: 'etched', name: 'Etched' },
			{ key: 'glossy', name: 'Glossy' }
		]);
	});

	it('should handle empty finishes array', () => {
		const result = getAvailableTreatments([], ['showcase']);
		expect(result).toEqual([]);
	});

	it('should handle undefined frame effects', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Nonfoil' },
			{ key: 'foil', name: 'Foil' }
		]);
	});

	it('should preserve finish order', () => {
		const result = getAvailableTreatments(['foil', 'nonfoil', 'etched'], []);
		expect(result[0].key).toBe('foil');
		expect(result[1].key).toBe('nonfoil');
		expect(result[2].key).toBe('etched');
	});

	it('should handle complex real-world card', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil', 'etched'], ['showcase', 'surgefoil']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Showcase' },
			{ key: 'foil', name: 'Showcase - Surge Foil' },
			{ key: 'etched', name: 'Showcase - Surge Foil' }
		]);
	});

	it('should handle extended art variants', () => {
		const result = getAvailableTreatments(['nonfoil', 'foil'], ['extendedart']);
		expect(result).toEqual([
			{ key: 'nonfoil', name: 'Extended Art' },
			{ key: 'foil', name: 'Extended Art - Foil' }
		]);
	});
});
