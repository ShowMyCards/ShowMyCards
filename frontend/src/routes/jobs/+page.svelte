<script lang="ts">
	import {
		JobStatusPending,
		JobStatusInProgress,
		JobStatusCompleted,
		JobStatusFailed,
		JobStatusCancelled,
		PageHeader,
		StatsCard,
		TableCard,
		Lozenge,
		Notification,
		EmptyState,
		Pagination,
		ScheduledTaskCard,
		Modal,
		type Job
	} from '$lib';
	import type { PageData } from './$types';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	let { data }: { data: PageData } = $props();

	let errorModalOpen = $state(false);
	let errorModalMessage = $state('');

	function getStatusColor(status: string): 'info' | 'warning' | 'success' | 'error' | null {
		if (status === JobStatusPending) return 'info';
		if (status === JobStatusInProgress) return 'warning';
		if (status === JobStatusCompleted) return 'success';
		if (status === JobStatusFailed) return 'error';
		return null;
	}

	function getStatusText(status: string) {
		if (status === JobStatusPending) return 'Pending';
		if (status === JobStatusInProgress) return 'In Progress';
		if (status === JobStatusCompleted) return 'Completed';
		if (status === JobStatusFailed) return 'Failed';
		if (status === JobStatusCancelled) return 'Cancelled';
		return status;
	}

	function formatDuration(job: Job): string {
		if (!job.started_at) return 'Not started';

		const start = new Date(job.started_at).getTime();
		const end = job.completed_at ? new Date(job.completed_at).getTime() : Date.now();
		const duration = Math.floor((end - start) / 1000);

		if (duration < 60) return `${duration}s`;
		if (duration < 3600) return `${Math.floor(duration / 60)}m ${duration % 60}s`;
		return `${Math.floor(duration / 3600)}h ${Math.floor((duration % 3600) / 60)}m`;
	}

	function parseJobMetadata(
		raw: string
	): { total_cards?: number; processed_cards?: number; phase?: string } | null {
		try {
			return JSON.parse(raw);
		} catch {
			return null;
		}
	}

	function handlePageChange(newPage: number) {
		goto(resolve(`/jobs?page=${newPage}` as '/jobs'));
	}

	// Calculate job statistics
	const jobStats = $derived.by(() => {
		const total = data.jobs.length;
		const completed = data.jobs.filter((j) => j.status === JobStatusCompleted).length;
		const failed = data.jobs.filter((j) => j.status === JobStatusFailed).length;
		const inProgress = data.jobs.filter((j) => j.status === JobStatusInProgress).length;
		const pending = data.jobs.filter((j) => j.status === JobStatusPending).length;

		return [
			{
				title: 'Total Jobs',
				value: total,
				description: 'On this page'
			},
			{
				title: 'Completed',
				value: completed,
				description: total > 0 ? `${Math.round((completed / total) * 100)}% success` : '0%',
				valueClass: 'text-success'
			},
			{
				title: 'Failed',
				value: failed,
				description: total > 0 ? `${Math.round((failed / total) * 100)}% failure` : '0%',
				valueClass: 'text-error'
			},
			{
				title: 'In Progress',
				value: inProgress,
				description: pending > 0 ? `${pending} pending` : 'None pending',
				valueClass: 'text-warning'
			}
		];
	});
</script>

<svelte:head>
	<title>Jobs - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader
		title="Jobs & Scheduled Tasks"
		description="View scheduled tasks and background job history" />

	{#if data.error}
		<Notification type="error">
			{data.error}
		</Notification>
	{/if}

	{#if data.scheduledTasks && data.scheduledTasks.length > 0}
		<div class="mb-8">
			<h2 class="text-2xl font-bold mb-4">Scheduled Tasks</h2>
			<div class="grid grid-cols-1 md:grid-cols-2 gap-4 mb-8">
				{#each data.scheduledTasks as task (task.name)}
					<ScheduledTaskCard {task} />
				{/each}
			</div>
		</div>

		<div class="divider"></div>
	{/if}

	<h2 class="text-2xl font-bold mb-4">Job History</h2>

	{#if data.jobs.length === 0}
		<EmptyState message="No jobs found" />
	{:else}
		<StatsCard stats={jobStats} class="mb-6 w-full" />

		<TableCard>
			<table class="table table-zebra">
				<thead>
					<tr>
						<th>ID</th>
						<th>Type</th>
						<th>Status</th>
						<th>Started</th>
						<th>Duration</th>
						<th>Progress</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>
					{#each data.jobs as job (job.id)}
						<tr>
							<td>{job.id}</td>
							<td>
								<!-- <span class="badge badge-outline">
									{job.type === 'bulk_data_import' ? 'Bulk Data Import' : job.type}
								</span> -->
								<Lozenge style="outline">
									{job.type === 'bulk_data_import' ? 'Bulk Data Import' : job.type}
								</Lozenge>
							</td>
							<td>
								<Lozenge
									color={getStatusColor(job.status)}
									style={job.status === JobStatusCancelled ? 'ghost' : null}>
									{getStatusText(job.status)}
								</Lozenge>
							</td>
							<td>
								{#if job.started_at}
									{new Date(job.started_at).toLocaleString()}
								{:else}
									<span class="opacity-50">-</span>
								{/if}
							</td>
							<td>{formatDuration(job)}</td>
							<td>
								{#if job.metadata}
									{@const metadata = parseJobMetadata(job.metadata)}
									{#if metadata && (metadata.total_cards || metadata.processed_cards)}
										<div class="text-xs">
											{metadata.processed_cards || 0} / {metadata.total_cards || 0}
											{#if metadata.total_cards && metadata.total_cards > 0}
												({Math.round(
													((metadata.processed_cards || 0) / metadata.total_cards) * 100
												)}%)
											{/if}
										</div>
										{#if metadata.phase}
											<div class="text-xs opacity-70">{metadata.phase}</div>
										{/if}
									{:else}
										<span class="opacity-50">-</span>
									{/if}
								{:else}
									<span class="opacity-50">-</span>
								{/if}
							</td>
							<td>
								{#if job.error}
									<button
										class="btn btn-ghost btn-xs"
										onclick={() => {
											errorModalMessage = job.error || 'Unknown error';
											errorModalOpen = true;
										}}>
										View Error
									</button>
								{:else}
									<span class="opacity-50">-</span>
								{/if}
							</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</TableCard>

		<Pagination
			currentPage={data.page}
			totalPages={data.totalPages}
			onPageChange={handlePageChange} />
	{/if}
</div>

<!-- Error Details Modal -->
<Modal open={errorModalOpen} onClose={() => (errorModalOpen = false)} title="Job Error Details">
	<pre class="bg-base-200 p-4 rounded overflow-x-auto text-sm">{errorModalMessage}</pre>
	<div class="modal-action">
		<button type="button" onclick={() => (errorModalOpen = false)} class="btn btn-ghost">
			Close
		</button>
	</div>
</Modal>
