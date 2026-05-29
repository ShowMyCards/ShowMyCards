import type { CardResult } from '../types/api';

/**
 * Returns the card's printed (localized) name when available, otherwise the
 * canonical English name. Scryfall only sets `printed_name` for non-English
 * single-faced cards, so English cards and multi-faced cards fall back to `name`.
 */
export function getDisplayName(card: Pick<CardResult, 'name' | 'printed_name'>): string {
	return card.printed_name ?? card.name;
}
