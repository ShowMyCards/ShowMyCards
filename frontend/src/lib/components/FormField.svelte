<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		label: string;
		id: string;
		name?: string;
		type?: string;
		placeholder?: string;
		value?: string | number;
		required?: boolean;
		disabled?: boolean;
		helper?: string;
		error?: string;
		inputClass?: string;
		children?: Snippet;
		// Validation props
		min?: number | string;
		max?: number | string;
		minlength?: number;
		maxlength?: number;
		pattern?: string;
		patternMessage?: string;
		validate?: (value: string | number) => string | undefined; // Custom validator
		validateOnBlur?: boolean; // When to trigger validation
	}

	let {
		label,
		id,
		name,
		type = 'text',
		placeholder,
		value = $bindable(),
		required = false,
		disabled = false,
		helper,
		error = $bindable(),
		inputClass,
		children,
		min,
		max,
		minlength,
		maxlength,
		pattern,
		patternMessage,
		validate,
		validateOnBlur = true
	}: Props = $props();

	let touched = $state(false);

	function handleBlur() {
		touched = true;
		if (validateOnBlur && validate) {
			const validationError = validate(value || '');
			error = validationError || undefined;
		}
	}

	function handleInput() {
		// Clear error on input if field was touched
		if (touched && error) {
			error = undefined;
		}

		// Run custom validation if not waiting for blur
		if (!validateOnBlur && validate) {
			const validationError = validate(value || '');
			error = validationError || undefined;
		}
	}

	// ARIA IDs for accessibility (derived to avoid Svelte warning)
	const helperId = $derived(`${id}-helper`);
	const errorId = $derived(`${id}-error`);
</script>

<div class="form-control">
	<div class="flex flex-col gap-2">
		<label for={id} class="font-semibold">
			{label}{required ? ' *' : ''}
		</label>

		{#if helper && !error}
			<p id={helperId} class="text-sm opacity-70">{helper}</p>
		{/if}

		{#if children}
			{@render children()}
		{:else}
			<input
				{id}
				name={name || id}
				{type}
				{placeholder}
				{required}
				{disabled}
				{min}
				{max}
				{minlength}
				{maxlength}
				{pattern}
				bind:value
				oninput={handleInput}
				onblur={handleBlur}
				aria-invalid={error ? 'true' : 'false'}
				aria-describedby={error ? errorId : helper ? helperId : undefined}
				class="input input-bordered {error ? 'input-error' : ''} {inputClass || ''}" />
		{/if}

		{#if error}
			<p id={errorId} class="text-sm text-error" role="alert">
				{error}
			</p>
		{/if}
	</div>
</div>
