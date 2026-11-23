<script lang="ts">
	import { browser } from '$app/env';
	import { PageHeader, notifications, getActionError, dataApi, SCRYFALL_LANGUAGES } from '#lib';
	import type { ExportData, ImportResponse } from '#lib';
	import SettingRow from '#lib/components/SettingRow.svelte';
	import SettingActions from '#lib/components/SettingActions.svelte';
	import DisplaySettings from '#lib/components/DisplaySettings.svelte';
	import type { PageData } from './$types';
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import { Download, Upload } from '@lucide/svelte';

	let { data }: { data: PageData } = $props();

	// eslint-disable-next-line svelte/prefer-writable-derived -- settings is modified locally for form binding
	let settings = $state<Record<string, string>>({});
	let saving = $state(false);

	// Data import state
	let fileInput = $state<HTMLInputElement | null>(null);
	let importPreview = $state<ExportData | null>(null);
	let importResult = $state<ImportResponse | null>(null);
	let importing = $state(false);
	let importError = $state<string | null>(null);

	// Only the keys this page edits are sent on save, so saving can't overwrite
	// values changed elsewhere (task schedules, last-run times written by the
	// scheduler) with the stale copy loaded here.
	const SCRYFALL_KEYS = [
		'scryfall_default_search',
		'scryfall_unique_mode',
		'scryfall_default_language'
	] as const;
	const scryfallSettings = $derived(
		Object.fromEntries(SCRYFALL_KEYS.map((key) => [key, settings[key] ?? '']))
	);

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

	async function handleFileSelect(event: Event) {
		const input = event.target as HTMLInputElement;
		const file = input.files?.[0];
		if (!file) return;

		importResult = null;
		importError = null;

		try {
			const text = await file.text();
			const parsed = JSON.parse(text);
			if (typeof parsed !== 'object' || parsed === null || typeof parsed.version !== 'number') {
				importError =
					'The selected file does not appear to be a valid ShowMyCards export (missing or invalid version field).';
				importPreview = null;
				return;
			}
			// Validate top-level shape: arrays where expected
			const arrayFields = [
				'storage_locations',
				'sorting_rules',
				'inventory',
				'lists',
				'decks'
			] as const;
			for (const field of arrayFields) {
				if (parsed[field] !== undefined && !Array.isArray(parsed[field])) {
					importError = `Invalid export file: "${field}" must be an array.`;
					importPreview = null;
					return;
				}
			}
			importPreview = parsed as ExportData;
		} catch {
			importError = 'The selected file is not valid JSON.';
			importPreview = null;
		}
	}

	async function handleImport() {
		if (!importPreview) return;

		importing = true;
		importError = null;

		try {
			importResult = await dataApi.import(importPreview);
			importPreview = null;
			notifications.success('Data imported successfully!');
		} catch (err) {
			importError = err instanceof Error ? err.message : 'Import failed';
			notifications.error(importError);
		} finally {
			importing = false;
			if (fileInput) fileInput.value = '';
		}
	}

	function cancelImport() {
		importPreview = null;
		importError = null;
		if (fileInput) fileInput.value = '';
	}

	function handleSaveEnhance() {
		saving = true;
		return async ({
			result,
			update
		}: {
			result: { type: string; data?: Record<string, unknown> };
			update: () => Promise<void>;
		}) => {
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
		<form method="POST" action="?/save" use:enhance={handleSaveEnhance}>
			<input type="hidden" name="settings" value={JSON.stringify(scryfallSettings)} />
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

						<SettingRow
							label="Default Language"
							description="Card language used for searches. Can be overridden explicitly in the search.">
							<select
								id="scryfall_default_language"
								bind:value={settings.scryfall_default_language}
								class="select select-bordered w-full max-w-xs">
								{#each SCRYFALL_LANGUAGES as lang (lang.code)}
									<option value={lang.code}>{lang.name}</option>
								{/each}
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

		<!-- Scheduled tasks moved to the Jobs page -->
		<div class="card bg-base-200 shadow-lg">
			<div class="card-body">
				<h2 class="card-title">Scheduled Tasks</h2>
				<p class="text-sm opacity-70">
					Card data updates, backups and other scheduled tasks are configured and run from the
					<a href={resolve('/jobs')} class="link link-primary">Jobs page</a>.
				</p>
			</div>
		</div>

		<!-- Data Management (Export / Import) -->
		<div class="card bg-base-200 shadow-lg">
			<div class="card-body">
				<h2 class="card-title mb-2">Data Management</h2>
				<p class="text-sm opacity-70 mb-4">
					Export your collection, lists, and decks as a JSON file for backup or migration. Import a
					previously exported file to restore or merge data.
				</p>

				<div class="flex flex-wrap gap-3">
					<a href={resolve('api/data/export')} class="btn btn-primary" download
						><Download class="w-4 h-4" /> Export Data</a>

					<button class="btn btn-secondary" onclick={() => fileInput?.click()}>
						<Upload class="w-4 h-4" />
						Import Data
					</button>
					<input
						type="file"
						accept=".json"
						bind:this={fileInput}
						onchange={handleFileSelect}
						class="hidden" />
				</div>

				{#if importError}
					<div class="alert alert-error mt-4">
						<span>{importError}</span>
					</div>
				{/if}

				{#if importPreview}
					{@const hasData = !!(
						importPreview.storage_locations?.length ||
						importPreview.sorting_rules?.length ||
						importPreview.inventory?.length ||
						importPreview.lists?.length ||
						importPreview.decks?.length
					)}
					<div class="alert {hasData ? 'alert-info' : 'alert-warning'} mt-4">
						<div>
							{#if hasData}
								<p class="font-semibold mb-2">Ready to import:</p>
								<ul class="list-disc list-inside text-sm space-y-1">
									{#if importPreview.storage_locations?.length}
										<li>
											{importPreview.storage_locations.length} storage location{importPreview
												.storage_locations.length !== 1
												? 's'
												: ''}
										</li>
									{/if}
									{#if importPreview.sorting_rules?.length}
										<li>
											{importPreview.sorting_rules.length} sorting rule{importPreview.sorting_rules
												.length !== 1
												? 's'
												: ''}
										</li>
									{/if}
									{#if importPreview.inventory?.length}
										<li>
											{importPreview.inventory.length} inventory item{importPreview.inventory
												.length !== 1
												? 's'
												: ''}
										</li>
									{/if}
									{#if importPreview.lists?.length}
										<li>
											{importPreview.lists.length} list{importPreview.lists.length !== 1 ? 's' : ''} ({importPreview.lists.reduce(
												(sum, l) => sum + (l.items?.length ?? 0),
												0
											)} items total)
										</li>
									{/if}
									{#if importPreview.decks?.length}
										<li>
											{importPreview.decks.length} deck{importPreview.decks.length !== 1 ? 's' : ''} ({importPreview.decks.reduce(
												(sum, d) => sum + (d.items?.length ?? 0),
												0
											)} items total)
										</li>
									{/if}
								</ul>
								<p class="text-sm opacity-70 mt-2">
									Imported data will be added alongside your existing collection.
								</p>
								<div class="flex gap-2 mt-3">
									<button
										class="btn btn-primary btn-sm"
										disabled={importing}
										onclick={handleImport}>
										{#if importing}
											<span class="loading loading-spinner loading-sm"></span>
											Importing...
										{:else}
											Confirm Import
										{/if}
									</button>
									<button class="btn btn-ghost btn-sm" disabled={importing} onclick={cancelImport}>
										Cancel
									</button>
								</div>
							{:else}
								<p class="font-semibold">This export file contains no data.</p>
								<p class="text-sm opacity-70 mt-1">
									The file is a valid export but has no storage locations, inventory, lists, decks,
									or sorting rules to import.
								</p>
								<div class="mt-3">
									<button class="btn btn-ghost btn-sm" onclick={cancelImport}>Dismiss</button>
								</div>
							{/if}
						</div>
					</div>
				{/if}

				{#if importResult}
					<div class="alert alert-success mt-4">
						<div>
							<p class="font-semibold mb-2">Import complete:</p>
							<ul class="list-disc list-inside text-sm space-y-1">
								{#if importResult.storage_locations_created}
									<li>
										{importResult.storage_locations_created} storage location{importResult.storage_locations_created !==
										1
											? 's'
											: ''} created
									</li>
								{/if}
								{#if importResult.sorting_rules_created}
									<li>
										{importResult.sorting_rules_created} sorting rule{importResult.sorting_rules_created !==
										1
											? 's'
											: ''} created
									</li>
								{/if}
								{#if importResult.inventory_items_created}
									<li>
										{importResult.inventory_items_created} inventory item{importResult.inventory_items_created !==
										1
											? 's'
											: ''} created
									</li>
								{/if}
								{#if importResult.lists_created}
									<li>
										{importResult.lists_created} list{importResult.lists_created !== 1 ? 's' : ''} created
									</li>
								{/if}
								{#if importResult.list_items_created}
									<li>
										{importResult.list_items_created} list item{importResult.list_items_created !==
										1
											? 's'
											: ''} created
									</li>
								{/if}
								{#if importResult.decks_created}
									<li>
										{importResult.decks_created} deck{importResult.decks_created !== 1 ? 's' : ''} created
									</li>
								{/if}
								{#if importResult.deck_items_created}
									<li>
										{importResult.deck_items_created} deck item{importResult.deck_items_created !==
										1
											? 's'
											: ''} created
									</li>
								{/if}
							</ul>
							{#if importResult.warnings?.length}
								<details class="mt-2">
									<summary class="cursor-pointer text-sm font-medium">
										{importResult.warnings.length} warning{importResult.warnings.length !== 1
											? 's'
											: ''}
									</summary>
									<ul class="list-disc list-inside text-xs mt-1 space-y-0.5 opacity-80">
										{#each importResult.warnings as warning, i (i)}
											<li>{warning}</li>
										{/each}
									</ul>
								</details>
							{/if}
						</div>
					</div>
				{/if}
			</div>
		</div>

		<!-- Display Settings (client-only) -->
		<DisplaySettings />
	</div>
</div>
