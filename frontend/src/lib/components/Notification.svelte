<script lang="ts" module>
	const HEADINGS = {
		info: 'Heads up!',
		warning: 'Tread lightly...',
		error: 'An error has occurred',
		success: 'Great Success!'
	} as const;

	const STYLES = {
		info: 'alert-info',
		warning: 'alert-warning',
		error: 'alert-error',
		success: 'alert-success'
	} as const;
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		type?: 'info' | 'warning' | 'error' | 'success';
		children: Snippet;
	}

	let { type = 'info', children }: Props = $props();
</script>

<div class="alert {STYLES[type]} mb-6">
	<div>
		<p class="font-semibold">{HEADINGS[type]}</p>
		<div>{@render children()}</div>
	</div>
</div>
