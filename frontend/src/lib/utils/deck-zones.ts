import type { DeckZone } from '#lib';

/** Display label for every deck zone. Record<DeckZone, …> makes a new zone a compile error here. */
export const ZONE_LABELS: Record<DeckZone, string> = {
	command: 'Command',
	main: 'Main',
	side: 'Side',
	maybe: 'Maybe',
	other: 'Other'
};

/**
 * Zones a user can put a card in. Other is left out: it is reserved for tokens
 * and other extras, which the backend always stores there whatever was picked.
 */
export const PICKABLE_ZONES: readonly DeckZone[] = ['command', 'main', 'side', 'maybe'];

/** Comma-separated labels for a list of zones, e.g. "Main, Other". */
export function zonesLabel(zones: readonly DeckZone[]): string {
	return zones.map((z) => ZONE_LABELS[z]).join(', ');
}
