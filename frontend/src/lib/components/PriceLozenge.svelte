<script lang="ts">
	import { Lozenge, currency } from '$lib';
	import type { CardPrices } from '$lib';
	import { getTreatmentPrice } from '$lib/utils/card-prices';

	interface Props {
		treatment: string;
		prices: CardPrices;
	}

	let { treatment, prices }: Props = $props();

	const price = $derived(getTreatmentPrice(prices, treatment, currency.current));
</script>

<Lozenge color="success" size="small">{currency.symbol}{price}</Lozenge>
