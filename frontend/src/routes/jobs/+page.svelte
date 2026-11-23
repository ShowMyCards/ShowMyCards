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
		JobDetailModal,
		TIMEOUTS,
		formatDuration,
		jobProgress,
		jobTypeLabel,
		jobTypeLabels,
		parseJobMetadata,
		statusColor,
		statusText,
		type JobRecord
	} from '#lib';
	import type { PageData } from './$types';
	import { goto, invalidateAll } from '$app/navigation';
	import { resolve } from '$app/paths';

	let { data }: { data: PageData } = $props();

	// While any task is running, poll so its card and the history table update.
	const anyTaskRunning = $derived(
		(data.scheduledTasks ?? []).some((task) => {
			const status = task.last_job_status ?? task.last_status;
			return status === JobStatusPending || status === JobStatusInProgress;
		})
	);

	$effect(() => {
		if (!anyTaskRunning) return;
		const interval = setInterval(() => invalidateAll(), TIMEOUTS.POLLING_INTERVAL);
		return () => clearInterval(interval);
	});

	// Job type labels come from the scheduler registry (each task's display name).
	const typeLabels = $derived(jobTypeLabels(data.scheduledTasks ?? []));

	let selectedJob = $state<JobRecord | null>(null);

	function handlePageChange(newPage: number) {
		goto(resolve(`/jobs?page=${newPage}`));
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
					<ScheduledTaskCard {task} onUpdated={() => invalidateAll()} />
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
						{@const progress = jobProgress(parseJobMetadata(job.metadata))}
						<tr id="job-{job.id}" class="scroll-mt-20 target:bg-base-300">
							<td>{job.id}</td>
							<td>
								<Lozenge style="outline">{jobTypeLabel(job.type, typeLabels)}</Lozenge>
							</td>
							<td>
								<Lozenge
									color={statusColor(job.status)}
									style={job.status === JobStatusCancelled ? 'ghost' : null}>
									{statusText(job.status)}
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
								{#if progress}
									<div class="text-xs">
										{progress.processed} / {progress.total}
										{#if progress.percent !== null}({progress.percent}%){/if}
									</div>
									{#if progress.phase}
										<div class="text-xs opacity-70">{progress.phase}</div>
									{/if}
								{:else}
									<span class="opacity-50">-</span>
								{/if}
							</td>
							<td>
								<button
									type="button"
									class="btn btn-ghost btn-xs {job.error ? 'text-error' : ''}"
									aria-label="Details for job {job.id}"
									onclick={() => (selectedJob = job)}>
									{job.error ? 'View error' : 'Details'}
								</button>
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

<JobDetailModal
	job={selectedJob}
	typeLabel={selectedJob ? jobTypeLabel(selectedJob.type, typeLabels) : ''}
	onClose={() => (selectedJob = null)} />
