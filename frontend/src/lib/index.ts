// place files you want to import through the `$lib` alias in this folder.

export { BACKEND_URL } from './config';

// Constants
export { TIMEOUTS, PAGINATION, JOB_STATUS } from './constants';

// Stores
export { notifications } from './stores/notifications.svelte';
export type { NotificationMessage } from './stores/notifications.svelte';
export { keyboard } from './stores/keyboard.svelte';
export type { CardActions, KeyboardShortcut } from './stores/keyboard.svelte';
export { selection } from './stores/selection.svelte';
export { currency } from './stores/currency.svelte';
export type { Currency } from './stores/currency.svelte';

// API Client
export { apiClient, ApiError } from './api/client';

// API Resources
export { storageApi } from './api/resources/storage';
export { inventoryApi } from './api/resources/inventory';
export { listsApi } from './api/resources/lists';
export { rulesApi } from './api/resources/rules';
export { jobsApi } from './api/resources/jobs';
export { settingsApi } from './api/resources/settings';
export { searchApi } from './api/resources/search';

// Composables
export { useFormState } from './composables/useFormState.svelte';
export { useAsyncAction } from './composables/useAsyncAction.svelte';
export { usePolling } from './composables/usePolling.svelte';
export { useExpressionValidation } from './composables/useExpressionValidation.svelte';
export { useRuleCrud } from './composables/useRuleCrud.svelte';
export { usePersistedViewMode } from './composables/usePersistedViewMode.svelte';

// Components
export { default as CardResultCard } from './components/CardResultCard.svelte';
export { default as Lozenge } from './components/Lozenge.svelte';
export { default as Notification } from './components/Notification.svelte';
export { default as NotificationContainer } from './components/NotificationContainer.svelte';
export { default as Modal } from './components/Modal.svelte';
export { default as EmptyState } from './components/EmptyState.svelte';
export { default as Pagination } from './components/Pagination.svelte';
export { default as PageHeader } from './components/PageHeader.svelte';
export { default as StatsCard } from './components/StatsCard.svelte';
export { default as TableCard } from './components/TableCard.svelte';
export { default as FormField } from './components/FormField.svelte';
export { default as SettingRow } from './components/SettingRow.svelte';
export { default as SettingActions } from './components/SettingActions.svelte';
export { default as ExpressionHelper } from './components/ExpressionHelper.svelte';
export { default as ExpressionInput } from './components/ExpressionInput.svelte';
export { default as RuleFormModal } from './components/RuleFormModal.svelte';
export { default as RulesTable } from './components/RulesTable.svelte';
export { default as RuleTester } from './components/RuleTester.svelte';
export { default as SortableRuleRow } from './components/SortableRuleRow.svelte';
export { default as ScheduledTaskCard } from './components/ScheduledTaskCard.svelte';
export { default as ListCard } from './components/ListCard.svelte';
export { default as ListStatsBar } from './components/ListStatsBar.svelte';
export { default as ListCardGrid } from './components/ListCardGrid.svelte';
export { default as ListCardTable } from './components/ListCardTable.svelte';
export { default as ListSearchResults } from './components/ListSearchResults.svelte';
export { default as StorageLocationRow } from './components/StorageLocationRow.svelte';
export { default as StorageLocationDropdown } from './components/StorageLocationDropdown.svelte';
export { default as KeyboardShortcutsHelp } from './components/KeyboardShortcutsHelp.svelte';
export { default as BulkActionsBar } from './components/BulkActionsBar.svelte';
export { default as InventoryBrowser } from './components/InventoryBrowser.svelte';
export { default as ResortModal } from './components/ResortModal.svelte';
export { default as PrintingConflictModal } from './components/PrintingConflictModal.svelte';
export { default as CardImage } from './components/CardImage.svelte';
export { default as CardPreview } from './components/CardPreview.svelte';
export { default as TreatmentBadge } from './components/TreatmentBadge.svelte';

// Card Collection (unified multi-view component)
export {
	CardCollection,
	GridView,
	TableView,
	ViewToggle,
	PageSizeSelector,
	CardFilter,
	adaptInventoryCards,
	adaptSearchResults,
	adaptListItems,
	fromInventoryCard,
	fromSearchResult,
	fromListItem,
	isFoilTreatment
} from './components/card-collection';
export type {
	ViewMode,
	CardSourceType,
	DisplayableCard,
	PaginationState,
	TableColumn,
	CardCollectionProps
} from './components/card-collection';

// Re-export generated types from Go backend models
export type { BaseModel, StorageType, JobType, JobStatus, Card } from './types/models';

export {
	Box,
	Binder,
	JobTypeBulkDataImport,
	JobStatusPending,
	JobStatusInProgress,
	JobStatusCompleted,
	JobStatusFailed,
	JobStatusCancelled
} from './types/models';

// Re-export runtime-accurate types (flattened BaseModel)
export type {
	RuntimeStorageLocation as StorageLocation,
	RuntimeInventory as Inventory,
	RuntimeSortingRule as SortingRule,
	RuntimeJob as Job,
	ScryfallCard,
	ListItemData,
	SearchFormResult
} from './types/runtime';

// Re-export List types from models
export type { List, ListItem } from './types/models';

// Re-export runtime-accurate API types
export type {
	SearchResponse,
	CardResult,
	CardPrices,
	EnhancedCardResult,
	CardInventoryData,
	EvaluateResponse,
	EvaluateRequest,
	StorageLocationWithCount,
	InventoryCardsResponse
} from './types/api-runtime';

// Re-export list-related API types
export type {
	ListSummary,
	CreateListRequest,
	UpdateListRequest,
	EnrichedListItem,
	CreateListItemRequest,
	CreateItemsBatchRequest,
	UpdateListItemRequest
} from './types/api';

// Re-export scheduler and dashboard types from API
export type { ScheduledTaskInfo, DashboardStats } from './types/api';

// Utilities
export { getCardTreatmentName, getAvailableTreatments } from './utils/card-treatment';
export {
	sortBySetAndCollectorNumber,
	createSetCollectorComparator,
	sortCardsBySetAndCollector
} from './utils/sort-cards';
export type { SetSortable } from './utils/sort-cards';
export { getUniqueTreatments, groupCardsByTreatment } from './utils/search-results';
export type { CardWithTreatment, GroupedCardsResult } from './utils/search-results';
export { audio } from './utils/audio';
export { getActionError, getActionMessage } from './utils/form-actions';
export { scryfallImageUrl } from './utils/scryfall-image';
