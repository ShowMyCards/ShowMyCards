<script lang="ts">
	import { getCardTreatmentName } from '$lib/utils/card-treatment';
	import { isFoilTreatment } from '$lib/components/card-collection';

	interface Props {
		/** The treatment/finish value (e.g., 'foil', 'nonfoil', 'etched') */
		treatment: string;
		/** Available finishes for the card */
		finishes?: string[];
		/** Frame effects for display name generation (e.g., ['showcase', 'extendedart']) */
		frameEffects?: string[];
		/** Promo types for special foil variants (e.g., ['surgefoil', 'galaxyfoil']) */
		promoTypes?: string[];
		/** Size variant */
		size?: 'xs' | 'sm' | 'md';
		/** Additional CSS classes */
		class?: string;
	}

	let {
		treatment,
		finishes = [treatment],
		frameEffects = [],
		promoTypes = [],
		size = 'sm',
		class: className = ''
	}: Props = $props();

	const displayName = $derived(getCardTreatmentName(finishes, frameEffects, treatment, promoTypes));
	const isFoil = $derived(isFoilTreatment(treatment));

	const sizeClasses = {
		xs: 'text-xs px-1.5 py-0.5',
		sm: 'text-xs px-2 py-0.5',
		md: 'text-sm px-2.5 py-1'
	};
</script>

{#if isFoil}
	<!-- Foil treatments get rainbow gradient background -->
	<span
		class="inline-flex items-center font-medium rounded
			bg-linear-to-r from-yellow-200 via-amber-200 to-orange-200
			text-amber-900 border border-amber-300/50
			{sizeClasses[size]} {className}">
		{displayName}
	</span>
{:else}
	<!-- Non-foil treatments get neutral styling -->
	<span
		class="inline-flex items-center font-medium rounded
			bg-base-300 text-base-content/80 border border-base-content/10
			{sizeClasses[size]} {className}">
		{displayName}
	</span>
{/if}
