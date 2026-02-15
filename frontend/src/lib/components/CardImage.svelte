<script lang="ts">
	interface Props {
		src?: string;
		alt: string;
		isFoil?: boolean;
		class?: string;
	}

	let { src, alt, isFoil = false, class: className = '' }: Props = $props();
</script>

<div class="card-image-container {className}">
	{#if src}
		<img {src} {alt} loading="lazy" class="card-image" />
		{#if isFoil}
			<div class="foil-overlay" aria-hidden="true"></div>
		{/if}
	{:else}
		<div class="card-placeholder">
			<span class="opacity-50">No image</span>
		</div>
	{/if}
</div>

<style>
	.card-image-container {
		position: relative;
		display: block;
		overflow: hidden;
		border-radius: 4.75% / 3.5%;
	}

	.card-image {
		display: block;
		width: 100%;
		height: auto;
	}

	.card-placeholder {
		width: 100%;
		aspect-ratio: 5 / 7;
		background: oklch(0.3 0 0);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.foil-overlay {
		position: absolute;
		inset: 0;
		background: linear-gradient(
			135deg,
			rgba(234, 141, 102, 0.4) 0%,
			rgba(253, 239, 138, 0.4) 15%,
			rgba(139, 204, 147, 0.4) 30%,
			rgba(166, 220, 237, 0.4) 45%,
			rgba(111, 117, 170, 0.4) 60%,
			rgba(229, 153, 194, 0.4) 75%,
			rgba(234, 141, 102, 0.4) 100%
		);
		background-size: 200% 200%;
		mix-blend-mode: color-dodge;
		pointer-events: none;
		animation: foil-shimmer 3s ease-in-out infinite;
	}

	@keyframes foil-shimmer {
		0% {
			background-position: 0% 0%;
		}
		50% {
			background-position: 100% 100%;
		}
		100% {
			background-position: 0% 0%;
		}
	}
</style>
