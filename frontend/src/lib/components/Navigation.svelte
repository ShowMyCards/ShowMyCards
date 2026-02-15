<script lang="ts">
	import { menuItems } from '$lib/data/navigation';
</script>

<div class="drawer-side is-drawer-close:overflow-visible">
	<label for="drawer" aria-label="close sidebar" class="drawer-overlay"></label>
	<div
		class="flex min-h-full flex-col items-start bg-base-200 is-drawer-close:w-14 is-drawer-open:w-64">
		<!-- Sidebar content here -->
		<ul class="menu w-full grow">
			{#each menuItems as item}
				{@const Icon = item.icon}
				<li>
					<a
						href={item.href}
						class="is-drawer-close:tooltip is-drawer-close:tooltip-right"
						data-tip={item.name}>
						<Icon class="my-1.5 inline-block size-4" />
						<span class="is-drawer-close:hidden">{item.name}</span>
					</a>
				</li>
			{/each}
		</ul>
	</div>
</div>
