/**
 * Image versions Scryfall serves. All are WebP except `jpeg`, which Scryfall
 * keeps for legacy software. The old JPEG names (`small`, `normal`, `large`,
 * `art_crop`, `border_crop`) go away after 2027-01-01, and `png` may no longer
 * be hotlinked.
 *
 * - `thumb`: 146 × 204
 * - `grid`: 488 × 680
 * - `display`: 672 × 936
 * - `crop`: 480 × 680, border cropped off
 * - `art`: art only, size varies
 * - `jpeg`: 488 × 680
 */
export type ScryfallImageVersion = 'thumb' | 'grid' | 'display' | 'crop' | 'art' | 'jpeg';

/**
 * Construct a Scryfall card image URL from a Scryfall card ID.
 * Uses the redirect endpoint so no API key or pre-fetched data is needed.
 */
export function scryfallImageUrl(
	scryfallId: string,
	version: ScryfallImageVersion = 'grid'
): string {
	return `https://api.scryfall.com/cards/${scryfallId}?format=image&version=${version}`;
}
