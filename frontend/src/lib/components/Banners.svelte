<script lang="ts">
	import { bannersApi, usePolling, type Banner as BannerData } from '$lib';
	import { browser } from '$app/environment';
	import { slide } from 'svelte/transition';
	import Banner from './Banner.svelte';

	// Relaxed on purpose: this is a passive background poll on a local app.
	const POLL_INTERVAL_MS = 10_000;
	const DISMISSED_STORAGE_KEY = 'dismissedBanners';

	let banners = $state<BannerData[]>([]);
	let dismissedIds = $state<string[]>(loadDismissedIds());

	function loadDismissedIds(): string[] {
		if (!browser) return [];
		try {
			const raw = localStorage.getItem(DISMISSED_STORAGE_KEY);
			const parsed = raw ? JSON.parse(raw) : [];
			return Array.isArray(parsed) ? parsed : [];
		} catch {
			return [];
		}
	}

	function persistDismissedIds(ids: string[]) {
		if (!browser) return;
		try {
			localStorage.setItem(DISMISSED_STORAGE_KEY, JSON.stringify(ids));
		} catch {
			// localStorage unavailable (private mode / quota): dismissal just
			// won't persist across reloads, which is acceptable.
		}
	}

	function dismiss(id: string) {
		if (dismissedIds.includes(id)) return;
		dismissedIds = [...dismissedIds, id];
		persistDismissedIds(dismissedIds);
	}

	usePolling(
		async () => {
			try {
				banners = await bannersApi.list();
			} catch {
				// Swallow transient failures: usePolling stops permanently if the
				// fetcher throws. The previous banners are kept until the next poll.
				return;
			}
			// Drop dismissals for banners the server no longer reports, so
			// localStorage can't grow without bound.
			const stillPresent = dismissedIds.filter((id) => banners.some((b) => b.id === id));
			if (stillPresent.length !== dismissedIds.length) {
				dismissedIds = stillPresent;
				persistDismissedIds(stillPresent);
			}
		},
		{ interval: POLL_INTERVAL_MS, enabled: true }
	);

	const visibleBanners = $derived(banners.filter((banner) => !dismissedIds.includes(banner.id)));
</script>

{#each visibleBanners as banner (banner.id)}
	<div transition:slide={{ duration: 200 }}>
		<Banner {banner} ondismiss={dismiss} />
	</div>
{/each}
