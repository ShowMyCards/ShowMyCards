<script lang="ts">
	import { PanelRightClose } from '@lucide/svelte';
	import Search from './SearchControl.svelte';
	import ThemeChanger from './ThemeChanger.svelte';
</script>

<nav class="navbar w-full bg-base-300">
	<div class="navbar-start">
		<label for="drawer" aria-label="open sidebar" class="btn btn-square btn-ghost">
			<PanelRightClose class="my-1.5 inline-block size-4" />
		</label>
	</div>
	<div class="navbar-center">
		<Search />
	</div>
	<div class="navbar-end">
		<ThemeChanger />
	</div>
</nav>
