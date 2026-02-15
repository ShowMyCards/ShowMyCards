<script lang="ts">
	import type { Snippet } from 'svelte';
	import CardImage from './CardImage.svelte';

	interface Props {
		src?: string;
		alt: string;
		isFoil?: boolean;
		children: Snippet;
		position?: 'left' | 'right';
	}

	let { src, alt, isFoil = false, children, position = 'right' }: Props = $props();

	let showPreview = $state(false);
	let hoverTimeout: ReturnType<typeof setTimeout> | null = null;

	function handleMouseEnter() {
		hoverTimeout = setTimeout(() => {
			showPreview = true;
		}, 150);
	}

	function handleMouseLeave() {
		if (hoverTimeout) {
			clearTimeout(hoverTimeout);
			hoverTimeout = null;
		}
		showPreview = false;
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div
	class="card-preview-trigger"
	onmouseenter={handleMouseEnter}
	onmouseleave={handleMouseLeave}
	onfocus={handleMouseEnter}
	onblur={handleMouseLeave}>
	{@render children()}

	{#if showPreview && src}
		<div
			class="card-preview-popover"
			class:position-left={position === 'left'}
			class:position-right={position === 'right'}
			role="tooltip">
			<CardImage {src} {alt} {isFoil} class="preview-image" />
		</div>
	{/if}
</div>

<style>
	.card-preview-trigger {
		position: relative;
		display: inline-block;
	}

	.card-preview-popover {
		position: absolute;
		z-index: 50;
		top: 50%;
		transform: translateY(-50%);
		width: 200px;
		pointer-events: none;
		filter: drop-shadow(0 10px 15px rgba(0, 0, 0, 0.3));
		animation: fade-in 0.15s ease-out;
	}

	.card-preview-popover :global(.preview-image) {
		border-radius: 4.75% / 3.5%;
	}

	.position-left {
		right: calc(100% + 12px);
	}

	.position-right {
		left: calc(100% + 12px);
	}

	@keyframes fade-in {
		from {
			opacity: 0;
			transform: translateY(-50%) scale(0.95);
		}
		to {
			opacity: 1;
			transform: translateY(-50%) scale(1);
		}
	}

	/* Ensure popover doesn't go off-screen on small viewports */
	@media (max-width: 768px) {
		.card-preview-popover {
			display: none;
		}
	}
</style>
