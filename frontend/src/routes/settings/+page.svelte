<script lang="ts">
	import { browser } from '$app/environment';
	import {
		PageHeader,
		Lozenge,
		notifications,
		getActionError
	} from '$lib';
	import SettingRow from '$lib/components/SettingRow.svelte';
	import SettingActions from '$lib/components/SettingActions.svelte';
	import DisplaySettings from '$lib/components/DisplaySettings.svelte';
	import BulkImportCard from '$lib/components/BulkImportCard.svelte';
	import type { PageData, ActionData } from './$types';
	import { enhance } from '$app/forms';

	let { data, form }: { data: PageData; form: ActionData } = $props();

	let settings = $state<Record<string, string>>({});
	let saving = $state(false);

	// Sync settings when data changes
	$effect(() => {
		settings = structuredClone(data.settings || {});
	});

	// Display load error if present (browser only)
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	function handleSaveEnhance() {
		saving = true;
		return async ({ result, update }: { result: { type: string; data?: Record<string, unknown> }; update: () => Promise<void> }) => {
			saving = false;
			await update();
			if (result.type === 'success') {
				notifications.success('Settings saved successfully!');
			} else if (result.type === 'failure') {
				const errorMsg = getActionError(result.data, 'Failed to save settings');
				notifications.error(errorMsg);
			}
		};
	}
</script>

<svelte:head>
	<title>Settings - ShowMyCards</title>
</svelte:head>

<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader title="Settings" description="Configure your ShowMyCards application" />

	<div class="space-y-6">
		<!-- Scryfall Search Configuration -->
		<form
			method="POST"
			action="?/save"
			use:enhance={handleSaveEnhance}>
			<input type="hidden" name="settings" value={JSON.stringify(settings)} />
			<div class="card bg-base-200 shadow-lg">
				<div class="card-body">
					<h2 class="card-title mb-4">Scryfall Search Options</h2>

					<div class="divide-y divide-base-300">
						<SettingRow
							label="Default Search String"
							description="Additional search terms appended to all Scryfall searches (e.g., 'game:paper')">
							<input
								type="text"
								id="scryfall_default_search"
								bind:value={settings.scryfall_default_search}
								placeholder="game:paper"
								class="input input-bordered w-full max-w-xs" />
						</SettingRow>

						<SettingRow
							label="Unique Mode"
							description="How to handle duplicate cards in search results">
							<select
								id="scryfall_unique_mode"
								bind:value={settings.scryfall_unique_mode}
								class="select select-bordered w-full max-w-xs">
								<option value="cards">Cards - Remove duplicate cards (default)</option>
								<option value="art">Art - Remove duplicate artworks</option>
								<option value="prints">Prints - Show all printings</option>
							</select>
						</SettingRow>
					</div>

					<SettingActions>
						<button type="submit" disabled={saving} class="btn btn-primary">
							{#if saving}
								<span class="loading loading-spinner loading-sm"></span>
								Saving...
							{:else}
								Save Settings
							{/if}
						</button>
					</SettingActions>
				</div>
			</div>
		</form>

		<!-- Bulk Data Configuration -->
		<form
			method="POST"
			action="?/save"
			use:enhance={handleSaveEnhance}>
			<input type="hidden" name="settings" value={JSON.stringify(settings)} />
			<div class="card bg-base-200 shadow-lg">
				<div class="card-body">
					<div class="mb-4">
						<h2 class="card-title">Bulk Data Configuration</h2>
						{#if settings.bulk_data_last_update}
							<p class="text-sm opacity-70 mt-1">
								Last updated {new Date(settings.bulk_data_last_update).toLocaleString()}
								{#if settings.bulk_data_last_update_status === 'success'}
									<Lozenge color="success" size="xsmall">Success</Lozenge>
								{:else if settings.bulk_data_last_update_status === 'failed'}
									<Lozenge color="error" size="xsmall">Failed</Lozenge>
								{:else if settings.bulk_data_last_update_status === 'in_progress'}
									<Lozenge color="warning" size="xsmall">In Progress</Lozenge>
								{/if}
								· <a href="/jobs" class="link link-primary text-sm">View job history</a>
							</p>
						{/if}
					</div>

					<div class="divide-y divide-base-300">
						<SettingRow
							label="Automatic Updates"
							description="Automatically update card database daily">
							<input
								type="checkbox"
								class="toggle toggle-primary"
								checked={settings.bulk_data_auto_update === 'true'}
								onchange={() => {
									settings.bulk_data_auto_update =
										settings.bulk_data_auto_update === 'true' ? 'false' : 'true';
								}} />
						</SettingRow>

						<SettingRow label="Update Time" description="Time when daily updates should run">
							<input
								type="time"
								id="update_time"
								bind:value={settings.bulk_data_update_time}
								class="input input-bordered w-full max-w-xs" />
						</SettingRow>

						<SettingRow label="Bulk Data Source" description="URL for Scryfall bulk data">
							<input
								type="text"
								id="bulk_data_url"
								value={settings.bulk_data_url || ''}
								readonly
								class="input input-bordered w-full opacity-70" />
						</SettingRow>
					</div>

					<SettingActions>
						<button type="submit" disabled={saving} class="btn btn-primary">
							{#if saving}
								<span class="loading loading-spinner loading-sm"></span>
								Saving...
							{:else}
								Save Settings
							{/if}
						</button>
					</SettingActions>
				</div>
			</div>
		</form>

		<!-- Display Settings (client-only) -->
		<DisplaySettings />

		<!-- Manual Import Section -->
		<BulkImportCard />
	</div>
</div>
