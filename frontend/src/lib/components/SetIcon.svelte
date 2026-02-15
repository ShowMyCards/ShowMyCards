<script lang="ts" module>
	// Module-level: shared across all component instances
	const svgCache = new Map<string, string>();
	let gradientInjected = false;

	function injectFoilGradient() {
		if (gradientInjected || typeof document === 'undefined') return;
		gradientInjected = true;

		const svg = document.createElementNS('http://www.w3.org/2000/svg', 'svg');
		svg.setAttribute('width', '0');
		svg.setAttribute('height', '0');
		svg.style.position = 'absolute';
		svg.innerHTML = `
			<defs>
				<linearGradient id="foil-gradient" x1="0%" y1="0%" x2="100%" y2="100%">
					<stop offset="0%" stop-color="#ea8d66" />
					<stop offset="15%" stop-color="#ea8d66" />
					<stop offset="28%" stop-color="#fdef8a" />
					<stop offset="42%" stop-color="#8bcc93" />
					<stop offset="55%" stop-color="#a6dced" />
					<stop offset="68%" stop-color="#6f75aa" />
					<stop offset="84%" stop-color="#e599c2" />
					<stop offset="100%" stop-color="#e599c2" />
				</linearGradient>
			</defs>
		`;
		document.body.appendChild(svg);
	}

	const DANGEROUS_SVG_ELEMENTS = ['script', 'foreignobject', 'iframe', 'object', 'embed'];
	const EVENT_ATTR_PATTERN = /^on/i;

	function sanitizeSvg(raw: string): string | null {
		try {
			const parser = new DOMParser();
			const doc = parser.parseFromString(raw, 'image/svg+xml');

			const parserError = doc.querySelector('parsererror');
			if (parserError) return null;

			for (const tag of DANGEROUS_SVG_ELEMENTS) {
				for (const el of doc.querySelectorAll(tag)) {
					el.remove();
				}
			}

			for (const el of doc.querySelectorAll('*')) {
				for (const attr of [...el.attributes]) {
					if (EVENT_ATTR_PATTERN.test(attr.name) || attr.name === 'href') {
						if (attr.name === 'href' && el.tagName !== 'use') {
							el.removeAttribute(attr.name);
						} else if (attr.name !== 'href') {
							el.removeAttribute(attr.name);
						}
					}
				}
			}

			const svgEl = doc.documentElement;
			return svgEl.outerHTML;
		} catch {
			return null;
		}
	}
</script>

<script lang="ts">
	import { browser } from '$app/environment';
	import { onMount } from 'svelte';

	let { setCode, setName, rarity, isFoil } = $props();

	let svgContent = $state<string | null>(null);

	onMount(() => {
		injectFoilGradient();
	});

	$effect(() => {
		const code = setCode;
		if (!code || !browser) return;

		// Check cache first
		const cached = svgCache.get(code);
		if (cached) {
			svgContent = cached;
			return;
		}

		// Fetch, sanitize, and cache
		fetch(`/api/sets/${code}/icon`)
			.then((res) => (res.ok ? res.text() : null))
			.then((raw) => {
				if (raw) {
					const safe = sanitizeSvg(raw);
					if (safe) {
						svgCache.set(code, safe);
						svgContent = safe;
					}
				}
			})
			.catch(() => {
				svgContent = null;
			});
	});
</script>

<span
	class="set-icon"
	class:foil={isFoil}
	class:rarity-common={rarity === 'common'}
	class:rarity-uncommon={rarity === 'uncommon'}
	class:rarity-rare={rarity === 'rare'}
	class:rarity-mythic={rarity === 'mythic'}
	title={`${setName} (${isFoil ? 'Foil' : 'Non-foil'} ${rarity})`}
	role="img"
	aria-label={setName}>
	{#if svgContent}
		{@html svgContent}
	{/if}
</span>

<style>
	.set-icon {
		display: inline-block;
		width: 2em;
		height: 2em;
		vertical-align: middle;
	}

	.set-icon :global(svg) {
		width: 100%;
		height: 100%;
		fill: currentColor;
	}

	/* Rarity colors from keyrune.css */
	.rarity-common {
		color: #1a1718;
	}

	:global([data-theme='dark']) .rarity-common,
	:global(.dark) .rarity-common {
		color: oklch(0.9 0 0);
	}

	.rarity-uncommon {
		color: #707883;
	}

	.rarity-rare {
		color: #a58e4a;
	}

	.rarity-mythic {
		color: #bf4427;
	}

	/* Foil effect - rainbow gradient matching keyrune.css */
	.foil :global(svg) {
		fill: url(#foil-gradient);
	}
</style>
