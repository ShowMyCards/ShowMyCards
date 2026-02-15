<script lang="ts">
	import { page } from '$app/state';
	import { Notification } from '$lib';
	import { resolve } from '$app/paths';
	import { Home, ArrowLeft, RefreshCw, TriangleAlert } from '@lucide/svelte';

	const status = $derived(page.status);
	const errorMessage = $derived(page.error?.message || 'An unexpected error occurred');

	// Categorize errors and provide helpful information
	const errorInfo = $derived.by(() => {
		switch (status) {
			case 404:
				return {
					title: 'Page Not Found',
					description: 'The page you are looking for does not exist.',
					suggestion: 'Check the URL or return to the homepage.',
					icon: '🔍'
				};
			case 403:
				return {
					title: 'Access Forbidden',
					description: 'You do not have permission to access this resource.',
					suggestion: 'If you believe this is an error, please contact support.',
					icon: '🔒'
				};
			case 500:
				return {
					title: 'Internal Server Error',
					description: 'Something went wrong on our end.',
					suggestion: 'Please try again later or contact support if the problem persists.',
					icon: '⚠️'
				};
			case 503:
				return {
					title: 'Service Unavailable',
					description: 'The server is temporarily unable to handle your request.',
					suggestion: 'Please try again in a few moments.',
					icon: '🔧'
				};
			default:
				return {
					title: `Error ${status}`,
					description: errorMessage,
					suggestion: 'Please try again or return to the homepage.',
					icon: '❌'
				};
		}
	});

	function handleGoBack() {
		if (window.history.length > 1) {
			window.history.back();
		} else {
			window.location.href = '/';
		}
	}

	function handleRefresh() {
		window.location.reload();
	}
</script>

<svelte:head>
	<title>{errorInfo.title} - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<div class="min-h-[60vh] flex flex-col items-center justify-center text-center">
		<!-- Error Icon -->
		<div class="text-8xl mb-6">{errorInfo.icon}</div>

		<!-- Error Status Code -->
		<div class="text-6xl font-bold text-primary mb-4">{status}</div>

		<!-- Error Title -->
		<h1 class="text-3xl font-bold mb-4">{errorInfo.title}</h1>

		<!-- Error Description -->
		<p class="text-lg opacity-70 mb-2 max-w-md">{errorInfo.description}</p>

		<!-- Suggestion -->
		<p class="text-sm opacity-60 mb-8 max-w-md">{errorInfo.suggestion}</p>

		<!-- Error Message (if available and different from description) -->
		{#if errorMessage && errorMessage !== errorInfo.description}
			<div class="mb-8 w-full max-w-2xl">
				<Notification type="error">
					<div class="flex items-start gap-2">
						<TriangleAlert class="w-5 h-5 shrink-0 mt-0.5" />
						<div class="text-left">
							<div class="font-semibold mb-1">Error Details:</div>
							<div class="text-sm opacity-90">{errorMessage}</div>
						</div>
					</div>
				</Notification>
			</div>
		{/if}

		<!-- Action Buttons -->
		<div class="flex flex-wrap gap-4 justify-center">
			<button onclick={handleGoBack} class="btn btn-primary">
				<ArrowLeft class="w-4 h-4" />
				Go Back
			</button>

			<a href={resolve('/')} class="btn btn-outline">
				<Home class="w-4 h-4" />
				Home
			</a>

			{#if status >= 500}
				<button onclick={handleRefresh} class="btn btn-ghost">
					<RefreshCw class="w-4 h-4" />
					Try Again
				</button>
			{/if}
		</div>

		<!-- Additional Help for 404 -->
		{#if status === 404}
			<div class="mt-12 text-sm opacity-50 max-w-md">
				<p class="italic">
					"...Then, in answer to my query, through the 'Net I loved so dearly, came its answer, dark
					and dreary: Quoth the server, {status}."
				</p>
				<p class="mt-2 text-xs">
					— With apologies to Edgar Allan Poe (<a
						href="https://web.archive.org/web/20221210061626/http://bash.org/?120296"
						target="_blank"
						class="link">bash.org</a
					>)
				</p>
			</div>
		{/if}
	</div>
</div>
