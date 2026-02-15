<script lang="ts">
	import { Lozenge } from '$lib';
	import { resolve } from '$app/paths';
	import type { ScheduledTaskInfo } from '$lib';

	let { task }: { task: ScheduledTaskInfo } = $props();

	// Live countdown state
	let countdown = $state('');

	// Update countdown every second
	$effect(() => {
		function updateCountdown() {
			const now = Date.now();
			const next = new Date(task.next_run).getTime();
			const diff = next - now;

			if (diff <= 0) {
				countdown = 'Running soon...';
			} else {
				const days = Math.floor(diff / 86400000);
				const hours = Math.floor((diff % 86400000) / 3600000);
				const minutes = Math.floor((diff % 3600000) / 60000);

				if (days > 0) {
					countdown = `in ${days}d ${hours}h`;
				} else if (hours > 0) {
					countdown = `in ${hours}h ${minutes}m`;
				} else {
					countdown = `in ${minutes}m`;
				}
			}
		}

		// Initial update
		updateCountdown();

		// Update every second
		const interval = setInterval(updateCountdown, 1000);

		// Cleanup
		return () => clearInterval(interval);
	});

	function getStatusColor(status?: string): 'info' | 'warning' | 'success' | 'error' | null {
		if (!status) return null;
		if (status === 'pending') return 'info';
		if (status === 'in_progress') return 'warning';
		if (status === 'completed') return 'success';
		if (status === 'failed') return 'error';
		return null;
	}

	function getStatusText(status?: string): string {
		if (!status) return 'Unknown';
		if (status === 'pending') return 'Pending';
		if (status === 'in_progress') return 'In Progress';
		if (status === 'completed') return 'Completed';
		if (status === 'failed') return 'Failed';
		if (status === 'cancelled') return 'Cancelled';
		return status;
	}
</script>

<div class="card bg-base-100 shadow-md border border-base-300">
	<div class="card-body p-5">
		<div class="flex items-start justify-between mb-3">
			<div>
				<h3 class="card-title text-lg">{task.name}</h3>
				<p class="text-sm opacity-70">Schedule: {task.schedule}</p>
			</div>
			<div class="flex gap-2">
				{#if task.enabled}
					<Lozenge color="success" style="outline">Enabled</Lozenge>
				{:else}
					<Lozenge color="error" style="ghost">Disabled</Lozenge>
				{/if}
			</div>
		</div>

		<div class="divider my-2"></div>

		<div class="grid grid-cols-2 gap-4 text-sm">
			<div>
				<div class="opacity-70 mb-1">Next Run</div>
				{#if task.enabled}
					<div class="font-semibold">
						{new Date(task.next_run).toLocaleString()}
					</div>
					<div class="text-xs opacity-70 mt-1">{countdown}</div>
				{:else}
					<div class="opacity-50">Task disabled</div>
				{/if}
			</div>

			<div>
				<div class="opacity-70 mb-1">Last Run</div>
				{#if task.last_run}
					<div class="font-semibold">
						{new Date(task.last_run).toLocaleString()}
					</div>
					{#if task.last_job_status}
						<div class="mt-1">
							<Lozenge color={getStatusColor(task.last_job_status)} size="small">
								{getStatusText(task.last_job_status)}
							</Lozenge>
						</div>
					{/if}
				{:else}
					<div class="opacity-50">Never run</div>
				{/if}
			</div>
		</div>

		{#if task.last_job_id}
			<div class="mt-3">
				<a href={resolve(`/jobs#${task.last_job_id}` as '/jobs')} class="link link-primary text-sm"> View last job → </a>
			</div>
		{/if}
	</div>
</div>
