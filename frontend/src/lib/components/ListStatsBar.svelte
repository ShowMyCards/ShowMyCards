<script lang="ts">
	interface Props {
		totalWanted: number;
		totalCollected: number;
		collectedValue: number;
		remainingValue: number;
		completion: number;
	}

	let { totalWanted, totalCollected, collectedValue, remainingValue, completion }: Props = $props();
</script>

<div class="stats stats-vertical lg:stats-horizontal shadow mb-6 w-full bg-base-200">
	<div class="stat">
		<div class="stat-title">Cards Wanted</div>
		<div class="stat-value text-primary">{totalWanted}</div>
	</div>

	<div class="stat">
		<div class="stat-title">Cards Collected</div>
		<div class="stat-value text-success">{totalCollected}</div>
	</div>

	<div class="stat">
		<div class="stat-title">Value Collected</div>
		<div class="stat-value text-success">${collectedValue.toFixed(2)}</div>
		<div class="stat-desc">Total value of collected cards</div>
	</div>

	<div class="stat">
		<div class="stat-title">Value Remaining</div>
		<div class="stat-value text-warning">${remainingValue.toFixed(2)}</div>
		<div class="stat-desc">Total value still needed</div>
	</div>

	<div class="stat">
		<div class="stat-title">Completion</div>
		<div class="stat-value">{completion}%</div>
		<progress class="progress progress-success mt-2" value={totalCollected} max={totalWanted}
		></progress>
	</div>
</div>
