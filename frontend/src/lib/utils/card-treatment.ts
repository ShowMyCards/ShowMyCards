/**
 * Card Treatment Utilities
 *
 * Utilities for generating human-readable treatment names from Scryfall card data.
 */

/**
 * Special foil variants that can appear in frame_effects or promo_types
 */
const SPECIAL_FOILS = [
	'surgefoil',
	'galaxyfoil',
	'oilslickfoil',
	'confettifoil',
	'halofoil',
	'raisedfoil',
	'ripplefoil',
	'fracturefoil',
	'manafoil',
	'firstplacefoil',
	'dragonscalefoil',
	'singularityfoil',
	'cosmicfoil',
	'chocobofoil'
] as const;

/**
 * Frame style effects that indicate collection-relevant card variants
 * Note: 'inverted', 'legendary', 'enchantment', etc. are informational and excluded
 */
const STYLE_EFFECTS = ['showcase', 'extendedart', 'shatteredglass'] as const;

/**
 * Base finishes that should not be split
 */
const BASE_FINISHES = ['nonfoil', 'foil', 'etched', 'glossy'];

/**
 * Converts a lowercase string to title case
 * @param str - String to convert (e.g., "surgefoil")
 * @returns Title cased string with spaces (e.g., "Surge Foil")
 */
function toTitleCase(str: string): string {
	const lower = str.toLowerCase();

	// Don't split base finishes
	if (BASE_FINISHES.includes(lower)) {
		return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
	}

	// Work on lowercase version for consistent handling
	let withSpaces = lower;

	// Simple replacements for known compound words
	withSpaces = withSpaces
		.replace('extendedart', 'extended art')
		.replace('shatteredglass', 'shattered glass');

	// Regex only for *foil variants (surgefoil, galaxyfoil, halofoil, etc.)
	withSpaces = withSpaces.replace(/(\w+)foil$/, '$1 foil');

	// Capitalize first letter of each word
	return withSpaces
		.split(/\s+/)
		.filter((word) => word.length > 0)
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}

/**
 * Generates a human-readable treatment name from Scryfall card data
 *
 * @param finishes - Array of finishes from Scryfall (e.g., ["nonfoil", "foil"])
 * @param frameEffects - Array of frame_effects from Scryfall (e.g., ["showcase", "extendedart"])
 * @param selectedFinish - The specific finish selected (e.g., "foil")
 * @param promoTypes - Array of promo_types from Scryfall (e.g., ["surgefoil", "galaxyfoil"])
 * @returns Human-readable treatment name (e.g., "Showcase - Surge Foil")
 *
 * @example
 * ```typescript
 * // Card with showcase surgefoil (surgefoil in promo_types)
 * getCardTreatmentName(['foil'], ['showcase'], 'foil', ['surgefoil'])
 * // Returns: "Showcase - Surge Foil"
 *
 * // Card with extended art
 * getCardTreatmentName(['nonfoil'], ['extendedart'], 'nonfoil')
 * // Returns: "Extended Art"
 *
 * // Regular foil card
 * getCardTreatmentName(['nonfoil', 'foil'], [], 'foil')
 * // Returns: "Foil"
 *
 * // Regular nonfoil card
 * getCardTreatmentName(['nonfoil'], [], 'nonfoil')
 * // Returns: "Nonfoil"
 * ```
 */
export function getCardTreatmentName(
	finishes: string[],
	frameEffects: string[] = [],
	selectedFinish: string = 'nonfoil',
	promoTypes: string[] = []
): string {
	const parts: string[] = [];

	// Extract all style effects (showcase, extended art, etc.) in order
	const styleEffects = frameEffects.filter((effect) => {
		const lowerEffect = effect.toLowerCase();
		return STYLE_EFFECTS.some((se) => se === lowerEffect);
	});
	styleEffects.forEach((effect) => {
		parts.push(toTitleCase(effect));
	});

	// Extract special foil variant (only if a foil finish is selected)
	// Check both frame_effects and promo_types for special foil types
	const isFoilFinish = selectedFinish !== 'nonfoil';
	if (isFoilFinish) {
		// First check frame_effects for special foils
		let specialFoil = frameEffects.find((effect) => {
			const lowerEffect = effect.toLowerCase();
			return SPECIAL_FOILS.some((sf) => sf === lowerEffect);
		});

		// If not found in frame_effects, check promo_types
		if (!specialFoil) {
			specialFoil = promoTypes.find((promo) => {
				const lowerPromo = promo.toLowerCase();
				return SPECIAL_FOILS.some((sf) => sf === lowerPromo);
			});
		}

		if (specialFoil) {
			parts.push(toTitleCase(specialFoil));
		} else {
			// If no special foil variant, use the base finish
			parts.push(toTitleCase(selectedFinish));
		}
	}

	// If no special characteristics, return the base finish
	if (parts.length === 0) {
		return toTitleCase(selectedFinish);
	}

	return parts.join(' - ');
}

/**
 * Gets all available treatments for a card based on its finishes, frame effects, and promo types
 *
 * @param finishes - Array of finishes from Scryfall
 * @param frameEffects - Array of frame_effects from Scryfall
 * @param promoTypes - Array of promo_types from Scryfall
 * @returns Array of objects with treatment key and display name
 *
 * @example
 * ```typescript
 * getAvailableTreatments(['nonfoil', 'foil'], ['showcase'], ['surgefoil'])
 * // Returns: [
 * //   { key: 'nonfoil', name: 'Showcase' },
 * //   { key: 'foil', name: 'Showcase - Surge Foil' }
 * // ]
 * ```
 */
export function getAvailableTreatments(
	finishes: string[],
	frameEffects: string[] = [],
	promoTypes: string[] = []
): Array<{ key: string; name: string }> {
	return finishes.map((finish) => ({
		key: finish,
		name: getCardTreatmentName(finishes, frameEffects, finish, promoTypes)
	}));
}
