import type { CreateDeckItemRequest, DeckZone } from '$lib';

/** A card offered to the "Add to deck" dialog. */
export interface AddToDeckCard {
	oracle_id: string;
	/** The printing the card came from; only sent when the user pins printings. */
	scryfall_id: string;
	name: string;
	/** Copies available to add (e.g. the selected inventory quantity). */
	copies: number;
	/** Display only: shown in the single-card dialog summary. */
	set_name?: string;
	image_uri?: string;
}

export interface BuildDeckItemsOptions {
	zone: DeckZone;
	/** Require the exact printing instead of any printing of the card. */
	pin: boolean;
	/** 'copies' sums each group's copies; a number sets that quantity per group. */
	quantity: 'copies' | number;
}

/**
 * Turn dialog cards into deck items, one per deck identity.
 *
 * Deck items are unique on (oracle, printing, treatment, zone). Unpinned items
 * have no printing, so two printings of the same card collapse into a single
 * Oracle-level item; pinned items stay per printing. Grouping here also keeps
 * a single batch free of duplicate keys, which the backend would reject.
 *
 * Treatment is left blank (any finish), matching the deck page's add flow:
 * finish is not part of the M1 allocation math.
 */
export function buildDeckItems(
	cards: AddToDeckCard[],
	{ zone, pin, quantity }: BuildDeckItemsOptions
): CreateDeckItemRequest[] {
	const groups = new Map<string, CreateDeckItemRequest>();

	for (const card of cards) {
		const scryfallId = pin ? card.scryfall_id : '';
		const key = `${card.oracle_id}|${scryfallId}`;
		const existing = groups.get(key);

		if (existing) {
			if (quantity === 'copies') existing.desired_quantity += card.copies;
			continue;
		}

		groups.set(key, {
			oracle_id: card.oracle_id,
			scryfall_id: scryfallId,
			treatment: '',
			desired_quantity: quantity === 'copies' ? card.copies : quantity,
			zone
		});
	}

	return [...groups.values()].filter((item) => item.desired_quantity >= 1);
}
