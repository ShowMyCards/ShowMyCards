<script lang="ts">
	interface Props {
		pageSize: number;
		pageSizes?: number[];
		onPageSizeChange: (size: number) => void;
	}

	let { pageSize, pageSizes = [20, 50, 100], onPageSizeChange }: Props = $props();

	function handleChange(event: Event & { currentTarget: HTMLSelectElement }) {
		const newSize = parseInt(event.currentTarget.value, 10);
		if (!isNaN(newSize)) {
			onPageSizeChange(newSize);
		}
	}
</script>

<div class="flex items-center gap-2">
	<label for="page-size" class="text-sm opacity-70">Show:</label>
	<select
		id="page-size"
		class="select select-sm select-bordered"
		value={pageSize}
		onchange={handleChange}>
		{#each pageSizes as size (size)}
			<option value={size}>{size}</option>
		{/each}
	</select>
</div>
