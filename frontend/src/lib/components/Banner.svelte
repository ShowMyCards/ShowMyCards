<script lang="ts">
	import type { Banner } from '$lib';
	import { Info, TriangleAlert, CircleAlert, X } from '@lucide/svelte';

	interface Props {
		banner: Banner;
		/** Called with the banner id when the user dismisses a dismissible banner. */
		ondismiss?: (id: string) => void;
	}

	let { banner, ondismiss }: Props = $props();

	// role is severity-driven: info announces politely, warning/error assertively.
	const SEVERITY = {
		info: { alertClass: 'alert-info', icon: Info, role: 'status' },
		warning: { alertClass: 'alert-warning', icon: TriangleAlert, role: 'alert' },
		error: { alertClass: 'alert-error', icon: CircleAlert, role: 'alert' }
	} as const;

	const style = $derived(SEVERITY[banner.severity as keyof typeof SEVERITY] ?? SEVERITY.info);
	const Icon = $derived(style.icon);
</script>

<div role={style.role} class="alert {style.alertClass} alert-soft rounded-none">
	<Icon class="size-5 shrink-0" aria-hidden="true" />
	<span class="grow text-sm">{banner.message}</span>
	{#if banner.link}
		<!-- href is a backend-authored in-app path; resolve() only accepts statically known route ids -->
		<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
		<a href={banner.link.href} class="btn btn-ghost btn-sm">{banner.link.label}</a>
	{/if}
	{#if banner.dismissible}
		<button
			type="button"
			class="btn btn-ghost btn-sm btn-circle"
			aria-label="Dismiss"
			onclick={() => ondismiss?.(banner.id)}>
			<X class="size-4" aria-hidden="true" />
		</button>
	{/if}
</div>
