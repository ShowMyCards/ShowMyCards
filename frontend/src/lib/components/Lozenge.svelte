<script lang="ts" module>
	const SIZES = {
		xsmall: 'badge-xsmall',
		small: 'badge-small',
		medium: 'badge-medium',
		large: 'badge-large',
		xlarge: 'badge-xlarge'
	} as const;

	const COLORS = {
		primary: 'badge-primary',
		secondary: 'badge-secondary',
		accent: 'badge-accent',
		info: 'badge-info',
		success: 'badge-success',
		warning: 'badge-warning',
		error: 'badge-error'
	} as const;

	const STYLES = {
		outline: 'badge-outline',
		dash: 'badge-dash',
		ghost: 'badge-ghost',
		soft: 'badge-soft'
	} as const;
</script>

<script lang="ts">
	import type { Snippet } from 'svelte';

	interface Props {
		size?: 'xsmall' | 'small' | 'medium' | 'large' | 'xlarge';
		color?: 'primary' | 'secondary' | 'accent' | 'info' | 'success' | 'warning' | 'error' | null;
		style?: 'outline' | 'dash' | 'ghost' | 'soft' | null;
		children: Snippet;
	}

	let { size = 'medium', color, style, children }: Props = $props();

	const classes = $derived(
		`badge ${SIZES[size]}${color ? ` ${COLORS[color]}` : ''}${style ? ` ${STYLES[style]}` : ''}`
	);
</script>

<span class={classes}>
	{@render children()}
</span>
