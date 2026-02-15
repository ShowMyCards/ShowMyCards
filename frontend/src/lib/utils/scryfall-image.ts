/**
 * Construct a Scryfall card image URL from a Scryfall card ID.
 * Uses the redirect endpoint so no API key or pre-fetched data is needed.
 */
export function scryfallImageUrl(
	scryfallId: string,
	version: 'small' | 'normal' | 'large' | 'png' | 'art_crop' | 'border_crop' = 'normal'
): string {
	return `https://api.scryfall.com/cards/${scryfallId}?format=image&version=${version}`;
}
