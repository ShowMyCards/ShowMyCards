<script lang="ts">
	import { enhance } from '$app/forms';
	import {
		TIMEOUTS,
		JobStatusInProgress,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusCancelled,
		notifications,
		getActionError,
		getActionMessage,
		type JobType,
		type JobStatus
	} from '$lib';
	import SettingActions from './SettingActions.svelte';
	import { onMount } from 'svelte';

	interface JobResponse {
		id: number;
		created_at: string;
		updated_at: string;
		type: JobType;
		status: JobStatus;
		started_at?: string;
		completed_at?: string;
		error?: string;
		metadata?: string;
	}

	let importing = $state(false);
	let latestJob = $state<JobResponse | null>(null);
	let pollInterval: ReturnType<typeof setInterval> | null = null;

	async function fetchLatestJob() {
		try {
			const response = await fetch(`/api/jobs?type=bulk_data_import&page_size=1`);
			if (!response.ok) return;

			const result = await response.json();
			if (result.data && result.data.length > 0) {
				latestJob = result.data[0];
			}
		} catch {
			// Silently handle fetch errors
		}
	}

	async function checkAndStartPolling() {
		await fetchLatestJob();
		if (latestJob && latestJob.status === JobStatusInProgress) {
			startPolling();
		}
	}

	function startPolling() {
		if (pollInterval) return;

		pollInterval = setInterval(async () => {
			await fetchLatestJob();
			if (
				latestJob &&
				(latestJob.status === JobStatusCompleted ||
					latestJob.status === JobStatusFailed ||
					latestJob.status === JobStatusCancelled)
			) {
				stopPolling();
			}
		}, TIMEOUTS.POLLING_INTERVAL);
	}

	function stopPolling() {
		if (pollInterval) {
			clearInterval(pollInterval);
			pollInterval = null;
		}
	}

	onMount(() => {
		checkAndStartPolling();
		return () => stopPolling();
	});
</script>

<div class="card bg-base-200 shadow-lg">
	<div class="card-body">
		<h2 class="card-title mb-4">Manual Import</h2>

		<p class="text-sm opacity-70 mb-4">
			Manually trigger a bulk data import. This will download the latest card data from Scryfall
			and replace existing data. <a href="/jobs" class="link link-primary">View import history</a>
		</p>

		<SettingActions>
			<form
				method="POST"
				action="?/triggerImport"
				use:enhance={() => {
					importing = true;
					return async ({ result, update }) => {
						importing = false;
						await update();
						if (result.type === 'success') {
							notifications.success(
								getActionMessage(result.data, 'Import started successfully!')
							);
							startPolling();
						} else if (result.type === 'failure') {
							const errorMsg = getActionError(result.data, 'Failed to trigger import');
							notifications.error(errorMsg);
						}
					};
				}}>
				<button
					type="submit"
					disabled={importing || latestJob?.status === JobStatusInProgress}
					class="btn btn-secondary">
					{#if importing}
						<span class="loading loading-spinner loading-sm"></span>
						Triggering Import...
					{:else if latestJob?.status === JobStatusInProgress}
						<span class="loading loading-spinner loading-sm"></span>
						Import In Progress...
					{:else}
						Trigger Import Now
					{/if}
				</button>
			</form>
		</SettingActions>
	</div>
</div>
