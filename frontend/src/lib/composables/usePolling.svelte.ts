/**
 * Reusable polling hook with automatic cleanup
 *
 * Manages interval-based data fetching with start/stop controls and automatic
 * cleanup on component unmount.
 *
 * @example
 * ```ts
 * const polling = usePolling(
 *   async () => {
 *     const data = await jobsApi.get(jobId);
 *     updateJobState(data);
 *   },
 *   {
 *     interval: 3000,
 *     enabled: true,
 *     stopCondition: () => job.status === 'completed'
 *   }
 * );
 *
 * // Manually control polling
 * polling.start();
 * polling.stop();
 * polling.restart();
 * ```
 */
export function usePolling(
	fetcher: () => Promise<void>,
	options: {
		interval: number;
		enabled?: boolean;
		stopCondition?: () => boolean;
	}
) {
	let intervalId = $state<ReturnType<typeof setInterval> | null>(null);
	let isPolling = $derived(intervalId !== null);
	let error = $state<string | null>(null);

	/**
	 * Execute one polling iteration
	 */
	async function poll() {
		try {
			await fetcher();
			error = null;

			// Check stop condition
			if (options.stopCondition?.()) {
				stop();
			}
		} catch (e) {
			error = e instanceof Error ? e.message : 'Polling error';
			stop();
		}
	}

	/**
	 * Start polling
	 */
	function start() {
		if (intervalId) return;

		poll(); // Immediate first fetch
		intervalId = setInterval(poll, options.interval);
	}

	/**
	 * Stop polling
	 */
	function stop() {
		if (intervalId) {
			clearInterval(intervalId);
			intervalId = null;
		}
	}

	/**
	 * Restart polling (stop then start)
	 */
	function restart() {
		stop();
		start();
	}

	// Auto-start if enabled
	$effect(() => {
		if (options.enabled) {
			start();
		} else {
			stop();
		}
	});

	// Auto-cleanup on unmount
	$effect(() => {
		return () => stop();
	});

	return {
		get isPolling() {
			return isPolling;
		},
		get error() {
			return error;
		},
		start,
		stop,
		restart
	};
}
