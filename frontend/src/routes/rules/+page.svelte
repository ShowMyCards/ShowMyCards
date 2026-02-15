<script lang="ts">
	import { browser } from '$app/environment';
	import {
		PageHeader,
		EmptyState,
		Modal,
		RuleFormModal,
		RulesTable,
		RuleTester,
		notifications,
		useRuleCrud
	} from '$lib';
	import { Plus } from '@lucide/svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import type { PageData } from './$types';

	let { data }: { data: PageData } = $props();

	// State - use $derived for data-based values
	let rules = $derived(data.rules || []);
	let storageLocations = $derived(data.storageLocations || []);

	// Pagination
	let currentPage = $derived(data.pagination.page);
	let totalPages = $derived(data.pagination.total_pages);

	// Display load error if present (browser only)
	let hasShownLoadError = $state(false);
	$effect(() => {
		if (!browser || hasShownLoadError) return;
		if (data.error) {
			hasShownLoadError = true;
			notifications.error(data.error);
		}
	});

	// Filtering
	let filterEnabled = $state<boolean | null>(null);

	// CRUD operations
	const crud = useRuleCrud(() => rules);

	// Navigation handlers
	async function handlePageChange(page: number) {
		const url = new URL(window.location.href);
		url.searchParams.set('page', page.toString());
		if (filterEnabled !== null) {
			url.searchParams.set('enabled', filterEnabled.toString());
		}
		await goto(resolve(`${url.pathname}${url.search}` as '/rules'), { invalidateAll: true });
	}

	async function handleFilterChange(enabled: boolean | null) {
		filterEnabled = enabled;
		const url = new URL(window.location.href);
		url.searchParams.set('page', '1');
		if (enabled !== null) {
			url.searchParams.set('enabled', enabled.toString());
		} else {
			url.searchParams.delete('enabled');
		}
		await goto(resolve(`${url.pathname}${url.search}` as '/rules'), { invalidateAll: true });
	}
</script>

<div class="mx-auto px-4 py-8">
	<!-- Page Header -->
	<PageHeader
		title="Storage Rules"
		description="Automatically sort cards into storage locations based on conditions">
		{#snippet actions()}
			<button onclick={crud.openCreateModal} class="btn btn-primary">
				<Plus class="w-4 h-4" />
				New Rule
			</button>
		{/snippet}
	</PageHeader>

	<!-- Filter tabs -->
	<div class="tabs tabs-boxed">
		<button
			class="tab {filterEnabled === null ? 'tab-active' : ''}"
			onclick={() => handleFilterChange(null)}>
			All Rules
		</button>
		<button
			class="tab {filterEnabled === true ? 'tab-active' : ''}"
			onclick={() => handleFilterChange(true)}>
			Enabled
		</button>
		<button
			class="tab {filterEnabled === false ? 'tab-active' : ''}"
			onclick={() => handleFilterChange(false)}>
			Disabled
		</button>
	</div>

	<!-- Rules table and tester -->
	{#if rules.length === 0}
		<EmptyState message="No sorting rules yet">
			<button onclick={crud.openCreateModal} class="btn btn-primary">
				<Plus class="w-4 h-4" />
				Create Rule
			</button>
		</EmptyState>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<div class="lg:col-span-2">
				<RulesTable
					rules={crud.localRules}
					highlightedRuleId={crud.highlightedRuleId}
					{currentPage}
					{totalPages}
					onEdit={crud.openEditModal}
					onDelete={crud.openDeleteModal}
					onToggleEnabled={crud.handleToggleEnabled}
					onDrop={crud.handleDrop}
					onPageChange={handlePageChange} />
			</div>

			<div class="lg:col-span-1">
				<div class="lg:sticky lg:top-4">
					<RuleTester {rules} onRuleMatch={crud.handleRuleMatch} />
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- Create Rule Modal -->
<RuleFormModal
	open={crud.showCreateModal}
	mode="create"
	formState={crud.createForm}
	expressionValidation={crud.createExpressionValidation}
	{storageLocations}
	onClose={() => (crud.showCreateModal = false)}
	onSubmit={crud.handleCreateRule}
	onInsertExpression={(expr) => crud.insertExpression(expr, true)} />

<!-- Edit Rule Modal -->
<RuleFormModal
	open={crud.showEditModal}
	mode="edit"
	formState={crud.editForm}
	expressionValidation={crud.editExpressionValidation}
	{storageLocations}
	onClose={() => (crud.showEditModal = false)}
	onSubmit={crud.handleEditRule}
	onInsertExpression={(expr) => crud.insertExpression(expr, false)} />

<!-- Delete Rule Modal -->
<Modal open={crud.showDeleteModal} onClose={() => (crud.showDeleteModal = false)} title="Delete Storage Rule">
	<p>Are you sure you want to delete <strong>{crud.ruleToDelete?.name}</strong>?</p>
	<p class="text-sm opacity-70 mt-2">This action cannot be undone.</p>

	<div class="modal-action">
		<button onclick={() => (crud.showDeleteModal = false)} class="btn btn-ghost">Cancel</button>
		<button onclick={crud.handleDeleteRule} class="btn btn-error">Delete Rule</button>
	</div>
</Modal>
