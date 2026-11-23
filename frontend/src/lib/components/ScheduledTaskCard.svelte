<script lang="ts">
	import { Lozenge, notifications } from '#lib';
	import { resolve } from '$app/paths';
	import type { ScheduledTaskInfo } from '#lib';
	import { Play } from '@lucide/svelte';

	let {
		task,
		onStarted
	}: {
		task: ScheduledTaskInfo;
		/** Called after a run has been requested (successfully or not), e.g. to reload data. */
		onStarted?: () => void;
	} = $props();

	// Live countdown state
	let countdown = $state('');

	// Tasks that write Job records report the job's status; others (e.g. backups)
	// report the status persisted in settings.
	const lastStatus = $derived(task.last_job_status ?? task.last_status);
	const running = $derived(lastStatus === 'pending' || lastStatus === 'in_progress');

	let starting = $state(false);

	async function runNow() {
		starting = true;
		try {
			// Relative URL: proxied to the backend by the /api/[...path] server route.
			const response = await fetch(`/api/scheduler/tasks/${encodeURIComponent(task.name)}/run`, {
				method: 'POST'
			});
			if (response.ok) {
				notifications.success(`${task.display_name} started`);
			} else {
				const body: { error?: string } = await response.json().catch(() => ({}));
				notifications.error(body.error ?? `Failed to start ${task.display_name}`);
			}
		} catch {
			notifications.error(`Failed to start ${task.display_name}`);
		} finally {
			starting = false;
		}
		onStarted?.();
	}

	// Update countdown every second
	$effect(() => {
		const nextRun = task.next_run;
		if (!nextRun) {
			countdown = '';
			return;
		}

		function updateCountdown() {
			const now = Date.now();
			const next = new Date(nextRun!).getTime();
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
		if (status === 'completed' || status === 'success') return 'success';
		if (status === 'failed') return 'error';
		return null;
	}

	function getStatusText(status?: string): string {
		if (!status) return 'Unknown';
		if (status === 'pending') return 'Pending';
		if (status === 'in_progress') return 'In Progress';
		if (status === 'completed') return 'Completed';
		if (status === 'success') return 'Succeeded';
		if (status === 'failed') return 'Failed';
		if (status === 'cancelled') return 'Cancelled';
		return status;
	}
</script>

<div class="card bg-base-100 shadow-md border border-base-300">
	<div class="card-body p-5">
		<div class="flex items-start justify-between mb-3">
			<div>
				<h3 class="card-title text-lg">{task.display_name}</h3>
				{#if task.description}
					<p class="text-sm opacity-70">{task.description}</p>
				{/if}
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
				{#if !task.enabled}
					<div class="opacity-50">Task disabled</div>
				{:else if task.next_run}
					<div class="font-semibold">
						{new Date(task.next_run).toLocaleString()}
					</div>
					<div class="text-xs opacity-70 mt-1">{countdown}</div>
				{:else}
					<div class="opacity-50">Not scheduled</div>
				{/if}
			</div>

			<div>
				<div class="opacity-70 mb-1">Last Run</div>
				{#if task.last_run}
					<div class="font-semibold">
						{new Date(task.last_run).toLocaleString()}
					</div>
					{#if lastStatus}
						<div class="mt-1">
							<Lozenge color={getStatusColor(lastStatus)} size="small">
								{getStatusText(lastStatus)}
							</Lozenge>
						</div>
					{/if}
				{:else}
					<div class="opacity-50">Never run</div>
				{/if}
			</div>
		</div>

		<div class="mt-3 flex items-center justify-between gap-2">
			{#if task.last_job_id}
				<a
					href={resolve(`/jobs#job-${task.last_job_id}` as '/jobs')}
					class="link link-primary text-sm">
					View last job →
				</a>
			{:else}
				<span></span>
			{/if}
			<button
				type="button"
				class="btn btn-sm btn-outline"
				disabled={starting || running}
				title={running ? `${task.display_name} is already running` : undefined}
				onclick={runNow}>
				{#if starting}
					<span class="loading loading-spinner loading-xs"></span>
				{:else}
					<Play class="w-4 h-4" />
				{/if}
				Run now
			</button>
		</div>
	</div>
</div>
