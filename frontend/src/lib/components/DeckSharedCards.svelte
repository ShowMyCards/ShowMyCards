<script lang="ts">
	// Explains the deck's over-committed cards: one chip per card, and clicking a
	// chip shows why: the decks together want more copies than you own, or decks
	// pin one printing more often than you own it. Each reason lists the decks
	// behind it, so a conflict caused elsewhere is visible from every deck it flags.
	import type { AllocationConflict, SharedCard } from '#lib';
	import { getDisplayName, SCRYFALL_LANGUAGES } from '#lib';
	import { resolve } from '$app/paths';

	let { cards, deckId }: { cards: SharedCard[]; deckId: number } = $props();

	let openId = $state<string | null>(null);
	const open = $derived(cards.find((card) => card.oracle_id === openId));

	function toggle(oracleId: string) {
		openId = openId === oracleId ? null : oracleId;
	}

	// "ECL #271 · German": the printing's set, number and language.
	function printingLabel(conflict: AllocationConflict): string {
		const where = [
			conflict.set_code?.toUpperCase(),
			conflict.collector_number && `#${conflict.collector_number}`
		]
			.filter(Boolean)
			.join(' ');
		const lang =
			conflict.lang &&
			(SCRYFALL_LANGUAGES.find((l) => l.code === conflict.lang)?.name ??
				conflict.lang.toUpperCase());
		return [where || 'One printing', lang].filter(Boolean).join(' · ');
	}
</script>

<div class="flex flex-wrap gap-1.5">
	{#each cards as card (card.oracle_id)}
		<button
			type="button"
			class="btn btn-xs"
			class:btn-active={openId === card.oracle_id}
			aria-expanded={openId === card.oracle_id}
			onclick={() => toggle(card.oracle_id)}>{getDisplayName(card) || 'Unknown card'}</button>
	{/each}
</div>

{#if open}
	<ul class="mt-1 space-y-1.5 text-sm">
		{#each open.conflicts as conflict (conflict.scryfall_id)}
			<li>
				<div>
					{#if conflict.scryfall_id}
						<span class="font-semibold">{printingLabel(conflict)}</span> is pinned
						<span class="font-semibold">{conflict.wanted}×</span> in your decks, you own
						<span class="font-semibold">{conflict.owned}</span>.
					{:else}
						Your decks want <span class="font-semibold">{conflict.wanted}</span>, you own
						<span class="font-semibold">{conflict.owned}</span>.
					{/if}
				</div>
				<div class="flex flex-wrap items-baseline gap-x-1.5 text-xs opacity-80">
					{#each conflict.decks as deck, i (deck.deck_id)}
						{#if i > 0}<span class="opacity-60" aria-hidden="true">·</span>{/if}
						<span>
							{#if deck.deck_id === deckId}
								<span class="font-medium">this deck</span>
							{:else}
								<a href={resolve(`decks/${deck.deck_id}`)} class="link link-hover"
									>{deck.deck_name}</a>
							{/if}
							{deck.quantity}
						</span>
					{/each}
				</div>
			</li>
		{/each}
	</ul>
{/if}
