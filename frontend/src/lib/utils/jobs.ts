import type { JobStatus, JobType } from '../types/models';
import type { ScheduledTaskInfo } from '../types/api';

/**
 * A job as served by GET /api/jobs. The generated `Job` model nests
 * `BaseModel`, but the JSON response flattens it, so this mirrors the wire shape.
 */
export interface JobRecord {
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

export type JobMetadata = Record<string, unknown>;

export type StatusColor = 'info' | 'warning' | 'success' | 'error' | null;

/** Title-cases a snake_case identifier: "data_backup" → "Data Backup". */
export function humanizeKey(key: string): string {
	return key
		.split('_')
		.filter(Boolean)
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}

/**
 * Maps each job type to the display name of the scheduled task that creates it.
 * The scheduler registry is the source of truth for labels.
 */
export function jobTypeLabels(tasks: ScheduledTaskInfo[]): Map<string, string> {
	const labels = new Map<string, string>();
	for (const task of tasks) {
		if (task.job_type) labels.set(task.job_type, task.display_name);
	}
	return labels;
}

/** Friendly label for a job type, falling back to a humanized identifier. */
export function jobTypeLabel(type: string, labels: Map<string, string>): string {
	return labels.get(type) ?? humanizeKey(type);
}

/**
 * Colour for a job status, or for a task status ("success" is the task-level
 * equivalent of a completed job).
 */
export function statusColor(status?: string): StatusColor {
	switch (status) {
		case 'pending':
			return 'info';
		case 'in_progress':
			return 'warning';
		case 'completed':
		case 'success':
			return 'success';
		case 'failed':
			return 'error';
		default:
			return null;
	}
}

/** Label for a job or task status. */
export function statusText(status?: string): string {
	switch (status) {
		case undefined:
		case '':
			return 'Unknown';
		case 'success':
			return 'Succeeded';
		default:
			return humanizeKey(status);
	}
}

/** Parses a job's metadata JSON; returns null when absent, invalid, or not an object. */
export function parseJobMetadata(raw?: string): JobMetadata | null {
	if (!raw) return null;
	try {
		const parsed: unknown = JSON.parse(raw);
		return parsed && typeof parsed === 'object' && !Array.isArray(parsed)
			? (parsed as JobMetadata)
			: null;
	} catch {
		return null;
	}
}

/** Elapsed time of a job, measured to now while it is still running. */
export function formatDuration(
	job: Pick<JobRecord, 'started_at' | 'completed_at'>,
	now = Date.now()
): string {
	if (!job.started_at) return 'Not started';

	const start = new Date(job.started_at).getTime();
	const end = job.completed_at ? new Date(job.completed_at).getTime() : now;
	const seconds = Math.max(0, Math.floor((end - start) / 1000));

	if (seconds < 60) return `${seconds}s`;
	if (seconds < 3600) return `${Math.floor(seconds / 60)}m ${seconds % 60}s`;
	return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`;
}

/** Human-readable byte count: 1536 → "1.5 KB". */
export function formatBytes(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	const units = ['KB', 'MB', 'GB', 'TB'];
	let value = bytes / 1024;
	let unit = 0;
	while (value >= 1024 && unit < units.length - 1) {
		value /= 1024;
		unit++;
	}
	return `${value.toFixed(value < 10 ? 1 : 0)} ${units[unit]}`;
}

export interface MetadataEntry {
	key: string;
	label: string;
	value: string;
	/** True for structured values (arrays/objects) rendered as preformatted JSON. */
	structured: boolean;
}

/**
 * Turns job metadata into display rows without knowing the job type. Keys
 * ending in `_bytes` are formatted as sizes (and labelled without the suffix);
 * structured values are pretty-printed JSON.
 */
export function metadataEntries(metadata: JobMetadata): MetadataEntry[] {
	return Object.entries(metadata).map(([key, raw]) => {
		const isBytes = key.endsWith('_bytes') && typeof raw === 'number';
		const label = humanizeKey(isBytes ? key.slice(0, -'_bytes'.length) : key);

		if (isBytes) return { key, label, value: formatBytes(raw), structured: false };
		if (raw !== null && typeof raw === 'object') {
			return { key, label, value: JSON.stringify(raw, null, 2), structured: true };
		}
		if (typeof raw === 'number')
			return { key, label, value: raw.toLocaleString(), structured: false };
		return { key, label, value: String(raw), structured: false };
	});
}

/** Card-count progress recorded by import jobs, if the metadata has it. */
export function jobProgress(
	metadata: JobMetadata | null
): { processed: number; total: number; percent: number | null; phase?: string } | null {
	if (!metadata) return null;
	const processed = typeof metadata.processed_cards === 'number' ? metadata.processed_cards : 0;
	const total = typeof metadata.total_cards === 'number' ? metadata.total_cards : 0;
	if (!processed && !total) return null;
	return {
		processed,
		total,
		percent: total > 0 ? Math.round((processed / total) * 100) : null,
		phase: typeof metadata.phase === 'string' ? metadata.phase : undefined
	};
}
