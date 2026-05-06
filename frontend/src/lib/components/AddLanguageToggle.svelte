<script lang="ts">
	import { addLanguage, MTG_LANGUAGES, type AddLanguage } from '$lib';
</script>

<select
	class="select select-sm select-bordered"
	aria-label="Add language"
	value={addLanguage.current}
	onchange={(e) => addLanguage.set(e.currentTarget.value as AddLanguage)}>
	{#each MTG_LANGUAGES as { code, name } (code)}
		<option value={code}>{name} ({code.toUpperCase()})</option>
	{/each}
</select>
