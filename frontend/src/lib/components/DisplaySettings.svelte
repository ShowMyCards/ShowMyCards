<script lang="ts">
	import { browser } from '$app/environment';
	import { audio, currency, type Currency } from '$lib';
	import SettingRow from './SettingRow.svelte';

	// Sound settings (client-only, stored in localStorage)
	let soundEnabled = $state(true);
	let soundVolume = $state(50);

	// Currency setting (client-only, uses currency store)
	let selectedCurrency = $state<Currency>(currency.current);

	// Load sound settings from localStorage
	$effect(() => {
		if (!browser) return;
		const storedSoundEnabled = localStorage.getItem('soundEnabled');
		const storedSoundVolume = localStorage.getItem('soundVolume');
		if (storedSoundEnabled !== null) {
			soundEnabled = storedSoundEnabled === 'true';
			audio.setEnabled(soundEnabled);
		}
		if (storedSoundVolume !== null) {
			soundVolume = parseInt(storedSoundVolume, 10);
			audio.setVolume(soundVolume / 100);
		}
	});

	function handleSoundToggle() {
		soundEnabled = !soundEnabled;
		audio.setEnabled(soundEnabled);
		localStorage.setItem('soundEnabled', String(soundEnabled));
	}

	function handleVolumeChange(event: Event & { currentTarget: HTMLInputElement }) {
		soundVolume = parseInt(event.currentTarget.value, 10);
		audio.setVolume(soundVolume / 100);
		localStorage.setItem('soundVolume', String(soundVolume));
	}

	function testSound() {
		audio.play('match');
	}

	function handleCurrencyChange(event: Event & { currentTarget: HTMLSelectElement }) {
		const newCurrency = event.currentTarget.value as Currency;
		selectedCurrency = newCurrency;
		currency.set(newCurrency);
	}
</script>

<div class="card bg-base-200 shadow-lg">
	<div class="card-body">
		<h2 class="card-title mb-4">Display Settings</h2>

		<div class="divide-y divide-base-300">
			<SettingRow
				label="Price Currency"
				description="Choose which currency to display for card prices">
				<select
					class="select select-bordered w-full max-w-xs"
					value={selectedCurrency}
					onchange={handleCurrencyChange}>
					<option value="usd">USD ($)</option>
					<option value="eur">EUR (€)</option>
				</select>
			</SettingRow>

			<SettingRow
				label="Enable Sounds"
				description="Play sounds when sorting rules match cards">
				<input
					type="checkbox"
					class="toggle toggle-primary"
					checked={soundEnabled}
					onchange={handleSoundToggle} />
			</SettingRow>

			<SettingRow label="Volume" description="Adjust the volume of sound effects">
				<div class="flex items-center gap-3 w-full max-w-xs">
					<input
						type="range"
						min="0"
						max="100"
						value={soundVolume}
						oninput={handleVolumeChange}
						disabled={!soundEnabled}
						class="range range-primary range-sm flex-1" />
					<span class="text-sm w-10 text-right">{soundVolume}%</span>
				</div>
			</SettingRow>

			<SettingRow label="Test Sound" description="Click to hear the rule match sound">
				<button
					type="button"
					class="btn btn-sm btn-outline"
					disabled={!soundEnabled}
					onclick={testSound}>
					Play Test Sound
				</button>
			</SettingRow>
		</div>
	</div>
</div>
