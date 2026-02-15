<script lang="ts">
	interface Props {
		currentPage: number;
		totalPages: number;
		onPageChange: (page: number) => void;
	}

	let { currentPage, totalPages, onPageChange }: Props = $props();
</script>

{#if totalPages > 1}
	<div class="flex justify-center mt-6">
		<div class="join">
			<button
				class="join-item btn"
				disabled={currentPage <= 1}
				onclick={() => onPageChange(currentPage - 1)}>
				«
			</button>
			<button class="join-item btn">
				Page {currentPage} of {totalPages}
			</button>
			<button
				class="join-item btn"
				disabled={currentPage >= totalPages}
				onclick={() => onPageChange(currentPage + 1)}>
				»
			</button>
		</div>
	</div>
{/if}
