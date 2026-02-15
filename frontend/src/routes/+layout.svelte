<script lang="ts">
	import './layout.css';
	import favicon from '$lib/assets/favicon.svg';
	import Navbar from '$lib/components/Navbar.svelte';
	import Navigation from '$lib/components/Navigation.svelte';
	import { NotificationContainer, KeyboardShortcutsHelp, keyboard, audio } from '$lib';
	import { onMount } from 'svelte';

	let { children } = $props();

	onMount(() => {
		// Initialize keyboard shortcuts and audio
		keyboard.attach();
		audio.init();

		// Restore audio settings from localStorage
		const storedSoundEnabled = localStorage.getItem('soundEnabled');
		const storedSoundVolume = localStorage.getItem('soundVolume');
		if (storedSoundEnabled !== null) {
			audio.setEnabled(storedSoundEnabled === 'true');
		}
		if (storedSoundVolume !== null) {
			audio.setVolume(parseInt(storedSoundVolume, 10) / 100);
		}

		return () => {
			keyboard.detach();
		};
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="drawer lg:drawer-open">
	<input id="drawer" type="checkbox" class="drawer-toggle" />
	<div class="drawer-content">
		<Navbar />
		<div class="p-4">
			{@render children()}
		</div>
	</div>

	<Navigation />
</div>

<!-- Global notification container -->
<NotificationContainer />

<!-- Keyboard shortcuts help overlay -->
<KeyboardShortcutsHelp />
