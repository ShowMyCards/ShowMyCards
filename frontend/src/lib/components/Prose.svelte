<script lang="ts">
	let { children } = $props();
</script>

<div class="prose lg:prose-xl dark:prose-invert max-w-none">
	{@render children?.()}
</div>
