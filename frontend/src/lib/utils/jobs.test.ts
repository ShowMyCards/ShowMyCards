import { describe, expect, it } from 'vitest';
import type { ScheduledTaskInfo } from '../types/api';
import {
	formatBytes,
	formatDuration,
	humanizeKey,
	jobProgress,
	jobTypeLabel,
	jobTypeLabels,
	metadataEntries,
	parseJobMetadata,
	statusColor,
	statusText
} from './jobs';

function task(name: string, display_name: string, job_type?: string): ScheduledTaskInfo {
	return { name, display_name, job_type } as ScheduledTaskInfo;
}

describe('job type labels', () => {
	const labels = jobTypeLabels([
		task('bulk_data_update', 'Card Data Update', 'bulk_data_import'),
		task('data_backup', 'Data Backup', 'data_backup'),
		task('no_jobs', 'No Jobs')
	]);

	it('uses the display name of the task that creates the job type', () => {
		expect(jobTypeLabel('bulk_data_import', labels)).toBe('Card Data Update');
		expect(jobTypeLabel('data_backup', labels)).toBe('Data Backup');
	});

	it('humanizes job types no registered task creates', () => {
		expect(jobTypeLabel('symbol_data_import', labels)).toBe('Symbol Data Import');
	});
});

describe('statusColor / statusText', () => {
	it.each([
		['pending', 'info', 'Pending'],
		['in_progress', 'warning', 'In Progress'],
		['completed', 'success', 'Completed'],
		['success', 'success', 'Succeeded'],
		['failed', 'error', 'Failed'],
		['cancelled', null, 'Cancelled']
	])('%s', (status, color, text) => {
		expect(statusColor(status)).toBe(color);
		expect(statusText(status)).toBe(text);
	});

	it('handles a missing status', () => {
		expect(statusColor(undefined)).toBeNull();
		expect(statusText(undefined)).toBe('Unknown');
	});
});

describe('parseJobMetadata', () => {
	it('parses an object', () => {
		expect(parseJobMetadata('{"deleted":3}')).toEqual({ deleted: 3 });
	});

	it.each([undefined, '', 'not json', '[1,2]', 'null'])('returns null for %j', (raw) => {
		expect(parseJobMetadata(raw)).toBeNull();
	});
});

describe('formatDuration', () => {
	const start = '2026-11-20T03:00:00Z';

	it.each([
		['2026-11-20T03:00:42Z', '42s'],
		['2026-11-20T03:02:05Z', '2m 5s'],
		['2026-11-20T04:30:00Z', '1h 30m']
	])('to %s', (completed_at, expected) => {
		expect(formatDuration({ started_at: start, completed_at })).toBe(expected);
	});

	it('measures to now while running', () => {
		expect(formatDuration({ started_at: start }, Date.parse('2026-11-20T03:00:10Z'))).toBe('10s');
	});

	it('reports jobs that have not started', () => {
		expect(formatDuration({})).toBe('Not started');
	});
});

describe('formatBytes', () => {
	it.each([
		[512, '512 B'],
		[1536, '1.5 KB'],
		[5 * 1024 * 1024, '5.0 MB'],
		[250 * 1024 * 1024, '250 MB']
	])('%i', (bytes, expected) => {
		expect(formatBytes(bytes)).toBe(expected);
	});
});

describe('humanizeKey', () => {
	it('title-cases snake_case', () => {
		expect(humanizeKey('retention_days')).toBe('Retention Days');
	});
});

describe('metadataEntries', () => {
	it('renders any metadata generically', () => {
		const entries = metadataEntries({
			file_name: 'showmycards-backup.json.gz',
			size_bytes: 2048,
			pruned: 1200,
			failure_examples: ['a', 'b'],
			deck_resort_error: 'database is locked'
		});

		expect(entries).toEqual([
			{
				key: 'file_name',
				label: 'File Name',
				value: 'showmycards-backup.json.gz',
				structured: false,
				error: false
			},
			{ key: 'size_bytes', label: 'Size', value: '2.0 KB', structured: false, error: false },
			{
				key: 'pruned',
				label: 'Pruned',
				value: (1200).toLocaleString(),
				structured: false,
				error: false
			},
			{
				key: 'failure_examples',
				label: 'Failure Examples',
				value: JSON.stringify(['a', 'b'], null, 2),
				structured: true,
				error: false
			},
			{
				key: 'deck_resort_error',
				label: 'Deck Resort Error',
				value: 'database is locked',
				structured: false,
				error: true
			}
		]);
	});

	it('does not flag empty _error strings as errors', () => {
		const [entry] = metadataEntries({ deck_resort_error: '' });
		expect(entry.error).toBe(false);
	});
});

describe('jobProgress', () => {
	it('reads card-count progress', () => {
		expect(jobProgress({ processed_cards: 50, total_cards: 200, phase: 'importing' })).toEqual({
			processed: 50,
			total: 200,
			percent: 25,
			phase: 'importing'
		});
	});

	it('returns null for metadata without progress', () => {
		expect(jobProgress({ file_name: 'x' })).toBeNull();
		expect(jobProgress(null)).toBeNull();
	});
});
