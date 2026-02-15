<script lang="ts">
	import { Modal, FormField, ExpressionInput, ExpressionHelper, type StorageLocation } from '$lib';
	import type { useFormState, useExpressionValidation } from '$lib';

	interface RuleFormValues extends Record<string, unknown> {
		name: string;
		expression: string;
		storage_location_id: number | '';
		enabled: boolean;
	}

	interface Props {
		open: boolean;
		mode: 'create' | 'edit';
		formState: ReturnType<typeof useFormState<RuleFormValues>>;
		expressionValidation: ReturnType<typeof useExpressionValidation>;
		storageLocations: StorageLocation[];
		onClose: () => void;
		onSubmit: () => void;
		onInsertExpression: (expression: string) => void;
	}

	let {
		open,
		mode,
		formState,
		expressionValidation,
		storageLocations,
		onClose,
		onSubmit,
		onInsertExpression
	}: Props = $props();

	const title = $derived(mode === 'create' ? 'Create Storage Rule' : 'Edit Storage Rule');
	const submitLabel = $derived(mode === 'create' ? 'Create Rule' : 'Update Rule');
	const nameId = $derived(mode === 'create' ? 'create-rule-name' : 'edit-rule-name');
	const expressionId = $derived(
		mode === 'create' ? 'create-rule-expression' : 'edit-rule-expression'
	);
	const storageId = $derived(
		mode === 'create' ? 'create-storage-location' : 'edit-storage-location'
	);

	function handleSubmit(e: Event) {
		e.preventDefault();
		onSubmit();
	}
</script>

<Modal {open} {onClose} {title}>
	<form onsubmit={handleSubmit} class="space-y-4">
		<div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
			<!-- Left Column: Form Fields -->
			<div class="space-y-4">
				<FormField
					label="Rule Name"
					id={nameId}
					name="name"
					placeholder="e.g., Expensive Mythics to Safe Storage"
					bind:value={formState.values.name}
					helper="A descriptive name for this rule"
					required />

				<ExpressionInput
					id={expressionId}
					bind:value={formState.values.expression}
					onchange={(val) => formState.setField('expression', val)}
					validation={expressionValidation}
					required />

				<div class="form-control">
					<label for={storageId} class="label">
						<span class="label-text">Storage Location <span class="text-error">*</span></span>
					</label>
					<select
						id={storageId}
						name="storage_location_id"
						bind:value={formState.values.storage_location_id}
						class="select select-bordered w-full"
						required>
						<option value="">Select a location...</option>
						{#each storageLocations as location (location.id)}
							<option value={location.id}>{location.name}</option>
						{/each}
					</select>
				</div>

				<div class="form-control">
					<label class="label cursor-pointer justify-start gap-2">
						<input
							type="checkbox"
							name="enabled"
							bind:checked={formState.values.enabled}
							class="toggle toggle-primary" />
						<span class="label-text">Enable this rule</span>
					</label>
				</div>
			</div>

			<!-- Right Column: Expression Helper -->
			<div class="hidden lg:block">
				<ExpressionHelper onInsert={onInsertExpression} />
			</div>
		</div>

		<!-- Mobile Expression Helper -->
		<div class="lg:hidden">
			<ExpressionHelper onInsert={onInsertExpression} />
		</div>

		<div class="modal-action">
			<button type="button" onclick={onClose} class="btn btn-ghost">Cancel</button>
			<button type="submit" class="btn btn-primary">{submitLabel}</button>
		</div>
	</form>
</Modal>
