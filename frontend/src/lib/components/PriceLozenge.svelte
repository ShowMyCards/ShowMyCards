<script lang="ts">
	import { Lozenge, currency } from '$lib';
	import type { CardPrices } from '$lib';

	interface Props {
		treatment: string;
		prices: CardPrices;
	}

	let { treatment, prices }: Props = $props();

	// Derive the price to display based on treatment and currency preference
	const priceData = $derived.by(() => {
		const isEur = currency.current === 'eur';
		const symbol = currency.symbol;

		let price: string | undefined;

		switch (treatment) {
			case 'foil':
				price = isEur ? prices.eur_foil : prices.usd_foil;
				break;
			case 'etched':
				// EUR doesn't have etched pricing, fall back to USD
				price = isEur ? prices.eur_foil : prices.usd_etched;
				break;
			default:
				price = isEur ? prices.eur : prices.usd;
		}

		return { price, symbol };
	});
</script>

<Lozenge color="success" size="small">{priceData.symbol}{priceData.price}</Lozenge>
