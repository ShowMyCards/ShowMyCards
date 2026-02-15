<script lang="ts">
	import { onMount } from 'svelte';
	import { menuItems } from '$lib/data/navigation';
	import { resolve } from '$app/paths';
	import { apiClient } from '$lib/api/client';

	let appVersion = $state('');

	onMount(async () => {
		try {
			const health = await apiClient.get<{ version: string }>('/health');
			appVersion = health.version;
		} catch {
			// Silently ignore — version display is non-critical
		}
	});
</script>

<div class="drawer-side is-drawer-close:overflow-visible">
	<label for="drawer" aria-label="close sidebar" class="drawer-overlay"></label>
	<div
		class="flex min-h-full flex-col items-start bg-base-200 is-drawer-close:w-14 is-drawer-open:w-64">
		<!-- Sidebar content here -->
		<ul class="menu w-full grow">
			{#each menuItems as item (item.href)}
				{@const Icon = item.icon}
				<li>
					<a
						href={resolve(item.href as '/')}
						class="is-drawer-close:tooltip is-drawer-close:tooltip-right"
						data-tip={item.name}>
						<Icon class="my-1.5 inline-block size-4" />
						<span class="is-drawer-close:hidden">{item.name}</span>
					</a>
				</li>
			{/each}
		</ul>
		{#if appVersion}
			<div class="w-full px-4 py-3 is-drawer-close:hidden">
				<span class="text-xs text-base-content/40">v{appVersion}</span>
			</div>
		{/if}
	</div>
</div>
