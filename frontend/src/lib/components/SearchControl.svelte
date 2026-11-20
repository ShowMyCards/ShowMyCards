<script lang="ts">
	import { resolve } from '$app/paths';
	import { SearchIcon } from '@lucide/svelte';
</script>

<form method="GET" action={resolve('search')}>
	<label class="input">
		<SearchIcon />
		<input type="text" name="q" placeholder="Search" aria-label="Search cards" class="w-full" />
	</label>
</form>
