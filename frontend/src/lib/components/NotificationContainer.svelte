<script lang="ts">
	import { notifications } from '../stores/notifications.svelte';
	import { fly } from 'svelte/transition';
	import { X } from '@lucide/svelte';

	// Get the notification list from the store
	const notificationList = $derived(notifications.notifications);

	// Map notification types to DaisyUI alert classes
	function getAlertClass(type: 'success' | 'error' | 'info' | 'warning'): string {
		switch (type) {
			case 'success':
				return 'alert-success';
			case 'error':
				return 'alert-error';
			case 'info':
				return 'alert-info';
			case 'warning':
				return 'alert-warning';
			default:
				return '';
		}
	}
</script>

<!-- Fixed position container at top-right -->
<div class="fixed top-4 right-4 z-50 flex flex-col gap-2 pointer-events-none max-w-md">
	{#each notificationList as notification (notification.id)}
		<div
			transition:fly={{ y: -20, duration: 300 }}
			class="alert {getAlertClass(notification.type)} shadow-lg pointer-events-auto">
			<span class="flex-1">{notification.message}</span>
			<button
				type="button"
				onclick={() => notifications.dismiss(notification.id)}
				class="btn btn-ghost btn-sm btn-circle"
				title="Dismiss notification"
				aria-label="Dismiss notification">
				<X class="w-4 h-4" />
			</button>
		</div>
	{/each}
</div>
