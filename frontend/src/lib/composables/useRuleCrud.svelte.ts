import { invalidateAll } from '$app/navigation';
import { deserialize } from '$app/forms';
import { notifications } from '../stores/notifications.svelte';
import { getActionError, getActionMessage } from '../utils/form-actions';
import { useFormState } from './useFormState.svelte';
import { useExpressionValidation } from './useExpressionValidation.svelte';
import { TIMEOUTS } from '../constants';
import type { RuntimeSortingRule as SortingRule } from '../types/runtime';
import type { DragDropState } from '@thisux/sveltednd';

/**
 * Submits a form action via fetch and handles success/failure notifications.
 */
async function submitAction(
	action: string,
	data: Record<string, string>,
	options: {
		successMessage: string;
		errorMessage: string;
		onSuccess?: () => void;
	}
) {
	const formData = new FormData();
	for (const [key, value] of Object.entries(data)) {
		formData.append(key, value);
	}

	try {
		const response = await fetch(`?/${action}`, {
			method: 'POST',
			body: formData
		});

		const result = deserialize(await response.text());

		if (result.type === 'success') {
			notifications.success(getActionMessage(result.data, options.successMessage));
			options.onSuccess?.();
			await invalidateAll();
			return true;
		} else if (result.type === 'failure') {
			notifications.error(getActionError(result.data, options.errorMessage));
		}
	} catch {
		notifications.error(options.errorMessage);
	}
	return false;
}

export function useRuleCrud(getRules: () => SortingRule[]) {
	// Modal state
	let showCreateModal = $state(false);
	let showEditModal = $state(false);
	let showDeleteModal = $state(false);

	// Form state - Create
	const createForm = useFormState({
		name: '',
		expression: '',
		storage_location_id: '' as number | '',
		enabled: true
	});

	// Form state - Edit
	let editRule = $state<SortingRule | null>(null);
	const editForm = useFormState({
		name: '',
		expression: '',
		storage_location_id: '' as number | '',
		enabled: true,
		priority: 1
	});

	// Delete state
	let ruleToDelete = $state<SortingRule | null>(null);

	// Expression validation
	const createExpressionValidation = useExpressionValidation({
		debounce: TIMEOUTS.VALIDATION_DEBOUNCE
	});
	const editExpressionValidation = useExpressionValidation({
		debounce: TIMEOUTS.VALIDATION_DEBOUNCE
	});

	// Rule highlighting
	let highlightedRuleId = $state<number | null>(null);

	// Local rules state for drag-and-drop (optimistic updates)
	let localRules = $state<SortingRule[]>([]);
	let isUpdatingPriorities = $state(false);

	// Sync localRules from server data when not updating priorities
	$effect.pre(() => {
		if (!isUpdatingPriorities) {
			localRules = getRules();
		}
	});

	function insertExpression(expression: string, isCreate: boolean) {
		if (isCreate) {
			createForm.setField('expression', expression);
			createExpressionValidation.setExpression(expression);
		} else {
			editForm.setField('expression', expression);
			editExpressionValidation.setExpression(expression);
		}
	}

	function openCreateModal() {
		createForm.reset();
		createExpressionValidation.setExpression('');
		showCreateModal = true;
	}

	async function handleCreateRule() {
		const values = createForm.values;

		if (!values.name.trim() || !values.expression.trim() || values.storage_location_id === '') {
			notifications.error('Please fill in all required fields');
			return;
		}

		if (!createExpressionValidation.isValid) {
			notifications.error('Please fix the expression error');
			return;
		}

		await submitAction(
			'create',
			{
				rule: JSON.stringify({
					name: values.name,
					expression: values.expression,
					storage_location_id: Number(values.storage_location_id),
					enabled: values.enabled,
					priority: getRules().length + 1
				})
			},
			{
				successMessage: 'Rule created successfully',
				errorMessage: 'Failed to create rule',
				onSuccess: () => {
					showCreateModal = false;
					createForm.reset();
					createExpressionValidation.setExpression('');
				}
			}
		);
	}

	function openEditModal(rule: SortingRule) {
		editRule = rule;
		editForm.setField('name', rule.name);
		editForm.setField('expression', rule.expression);
		editForm.setField('storage_location_id', rule.storage_location_id);
		editForm.setField('enabled', rule.enabled);
		editForm.setField('priority', rule.priority);
		editExpressionValidation.setExpression(rule.expression);
		showEditModal = true;
	}

	async function handleEditRule() {
		const values = editForm.values;

		if (
			!editRule ||
			!values.name.trim() ||
			!values.expression.trim() ||
			values.storage_location_id === ''
		) {
			notifications.error('Please fill in all required fields');
			return;
		}

		if (!editExpressionValidation.isValid) {
			notifications.error('Please fix the expression error');
			return;
		}

		await submitAction(
			'update',
			{
				id: String(editRule.id),
				rule: JSON.stringify({
					name: values.name,
					expression: values.expression,
					storage_location_id: Number(values.storage_location_id),
					enabled: values.enabled,
					priority: values.priority
				})
			},
			{
				successMessage: 'Rule updated successfully',
				errorMessage: 'Failed to update rule',
				onSuccess: () => {
					showEditModal = false;
				}
			}
		);
	}

	function openDeleteModal(rule: SortingRule) {
		ruleToDelete = rule;
		showDeleteModal = true;
	}

	async function handleDeleteRule() {
		if (!ruleToDelete) return;

		await submitAction(
			'delete',
			{ id: String(ruleToDelete.id) },
			{
				successMessage: 'Rule deleted successfully',
				errorMessage: 'Failed to delete rule',
				onSuccess: () => {
					showDeleteModal = false;
					ruleToDelete = null;
				}
			}
		);
	}

	async function handleToggleEnabled(rule: SortingRule) {
		await submitAction(
			'toggleEnabled',
			{
				id: String(rule.id),
				enabled: String(!rule.enabled)
			},
			{
				successMessage: `Rule ${!rule.enabled ? 'enabled' : 'disabled'}`,
				errorMessage: 'Failed to toggle rule'
			}
		);
	}

	function handleRuleMatch(ruleId: number) {
		highlightedRuleId = ruleId;

		setTimeout(() => {
			const ruleRow = document.querySelector(`[data-rule-id="${ruleId}"]`);
			if (ruleRow) {
				ruleRow.scrollIntoView({ behavior: 'smooth', block: 'center' });
			}
		}, 100);

		setTimeout(() => {
			highlightedRuleId = null;
		}, 3000);
	}

	function handleDrop(state: DragDropState<SortingRule>) {
		const { draggedItem, targetContainer } = state;
		if (!draggedItem || !targetContainer) return;

		const draggedRule = draggedItem;
		const targetIndex = parseInt(targetContainer);
		const draggedIndex = localRules.findIndex((r) => r.id === draggedRule.id);

		if (draggedIndex === -1 || draggedIndex === targetIndex) return;

		const newRules = [...localRules];
		const [removed] = newRules.splice(draggedIndex, 1);
		newRules.splice(targetIndex, 0, removed);

		const updated = newRules.map((rule, index) => ({
			...rule,
			priority: index + 1
		}));

		localRules = updated;
		savePriorities(updated);
	}

	async function savePriorities(updatedRules: SortingRule[]) {
		isUpdatingPriorities = true;

		const success = await submitAction(
			'batchUpdatePriorities',
			{
				updates: JSON.stringify(
					updatedRules.map((rule) => ({ id: rule.id, priority: rule.priority }))
				)
			},
			{
				successMessage: 'Priorities updated',
				errorMessage: 'Failed to update priorities'
			}
		);

		if (!success) {
			await invalidateAll();
		}

		isUpdatingPriorities = false;
	}

	return {
		// Modal state
		get showCreateModal() {
			return showCreateModal;
		},
		set showCreateModal(v: boolean) {
			showCreateModal = v;
		},
		get showEditModal() {
			return showEditModal;
		},
		set showEditModal(v: boolean) {
			showEditModal = v;
		},
		get showDeleteModal() {
			return showDeleteModal;
		},
		set showDeleteModal(v: boolean) {
			showDeleteModal = v;
		},

		// Form state
		createForm,
		editForm,
		createExpressionValidation,
		editExpressionValidation,

		// Delete/Edit targets
		get ruleToDelete() {
			return ruleToDelete;
		},

		// Drag-and-drop
		get localRules() {
			return localRules;
		},
		get highlightedRuleId() {
			return highlightedRuleId;
		},

		// Handlers
		openCreateModal,
		handleCreateRule,
		openEditModal,
		handleEditRule,
		openDeleteModal,
		handleDeleteRule,
		handleToggleEnabled,
		handleRuleMatch,
		handleDrop,
		insertExpression
	};
}
