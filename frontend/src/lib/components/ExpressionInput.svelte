<script lang="ts">
	import { useExpressionValidation } from '$lib';

	interface Props {
		value: string;
		onchange: (value: string) => void;
		validation: ReturnType<typeof useExpressionValidation>;
		label?: string;
		id?: string;
		placeholder?: string;
		required?: boolean;
		helper?: string;
	}

	let {
		value = $bindable(),
		onchange,
		validation,
		label = 'Expression',
		id = 'expression-input',
		placeholder = 'e.g., prices.usd > 10.0 && rarity == "mythic"',
		required = false,
		helper = 'Condition that cards must match (uses expr-lang syntax)'
	}: Props = $props();

	function handleInput(e: Event) {
		const newValue = (e.currentTarget as HTMLInputElement).value;
		value = newValue;
		validation.setExpression(newValue);
		onchange?.(newValue);
	}

	// Derive input class based on validation state
	const inputClass = $derived(
		validation.isValid ? 'input-success' : validation.error ? 'input-error' : ''
	);
</script>

<div class="form-control">
	<label for={id} class="label">
		<span class="label-text">{label}{required ? ' *' : ''}</span>
	</label>
	<div class="relative">
		<input
			{id}
			type="text"
			{value}
			oninput={handleInput}
			class="input input-bordered w-full {inputClass}"
			{placeholder}
			{required} />
		{#if validation.isValidating}
			<span class="absolute right-3 top-3 loading loading-spinner loading-sm"></span>
		{:else if validation.isValid}
			<span class="absolute right-3 top-3 text-success">✓</span>
		{:else if validation.error}
			<span class="absolute right-3 top-3 text-error">✗</span>
		{/if}
	</div>
	{#if validation.error}
		<div class="label">
			<span class="label-text-alt text-error">{validation.error}</span>
		</div>
	{/if}
	{#if helper}
		<div class="label">
			<span class="label-text-alt">{helper}</span>
		</div>
	{/if}
</div>
