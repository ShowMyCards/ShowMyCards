<script lang="ts">
	import { SearchIcon } from '@lucide/svelte';
</script>

<form method="POST" action="/search?/search">
	<label class="input">
		<SearchIcon />
		<input type="text" name="q" placeholder="Search" aria-label="Search cards" class="w-full" />
	</label>
</form>
