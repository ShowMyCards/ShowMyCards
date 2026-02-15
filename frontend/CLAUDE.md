# ShowMyCards Frontend

A modern web application for managing collectible card collections, built with SvelteKit and TypeScript.

This is part of the ShowMyCards monorepo. For development setup, Docker deployment, and build commands, see [DEVELOPMENT.md](../DEVELOPMENT.md) in the project root.

## Technology Stack

- **Framework**: SvelteKit 2.50+ (file-based routing, SSR/SSG)
- **UI Framework**: Svelte 5.50+ (modern reactive framework with runes)
- **Styling**: Tailwind CSS 4.1+ (utility-first CSS framework)
- **UI Components**: DaisyUI 5.5+ (Tailwind CSS component library)
- **Icons**: Lucide Svelte (SVG icon library)
- **Language**: TypeScript 5.9+
- **Build Tool**: Vite 7.3+ (fast dev server and bundler)
- **Testing**: Vitest 4.0+ (unit tests), Playwright 1.58+ (E2E tests)
- **Package Manager**: Bun (lockfile present)

## Directory Structure

```
ShowMyCards/
├── backend/                        # Go API server
├── frontend/                       # This directory - SvelteKit application
│   ├── src/
│   │   ├── routes/                 # File-based routing
│   │   │   ├── +layout.svelte      # Root layout with navigation
│   │   │   ├── +page.svelte        # Home/dashboard page
│   │   │   ├── +error.svelte       # Error page
│   │   │   ├── search/             # Card search
│   │   │   ├── storage/            # Storage locations
│   │   │   ├── inventory/          # Card inventory
│   │   │   ├── lists/              # Card lists (wishlists, decks)
│   │   │   ├── rules/              # Sorting rules
│   │   │   ├── jobs/               # Background jobs
│   │   │   └── settings/           # App settings
│   │   ├── lib/                    # Shared code
│   │   │   ├── components/         # Reusable UI components
│   │   │   ├── types/              # Auto-generated TypeScript types
│   │   │   │   ├── models.ts       # From backend/models/*.go
│   │   │   │   └── api.ts          # From backend/api/*.go
│   │   │   ├── utils/              # Utility functions
│   │   │   ├── config.ts           # App configuration (e.g., BACKEND_URL)
│   │   │   └── index.ts            # Library exports
│   │   ├── app.html                # HTML template
│   │   └── app.d.ts                # TypeScript definitions
│   ├── e2e/                        # End-to-end tests
│   ├── static/                     # Static assets (served as-is)
│   ├── svelte.config.js            # SvelteKit configuration
│   ├── vite.config.ts              # Vite + Vitest configuration
│   ├── playwright.config.ts        # Playwright E2E test config
│   └── package.json                # Dependencies and scripts
├── website/                        # Astro marketing site
├── docker/                         # Docker configurations
├── Makefile                        # Build commands
└── DEVELOPMENT.md                  # Development setup guide
```

## Code Reviews

All code reviews must follow FRONTEND_REVIEW_STANDARDS.md. Do not flag items listed in the Won't Fix section. Stop the review when only CONSIDER-level findings remain.

## Key Patterns

- **File-based routing**: Pages and layouts defined by file structure in `src/routes/`
- **Svelte 5 runes**: Modern reactivity with `$props()`, `$state()`, `$derived()`, etc.
- **Tailwind CSS 4**: Vite plugin integration for styling
- **Dual testing**: Vitest browser mode for component tests, Playwright for E2E
- **Test projects**: Separate client (browser) and server (node) test environments

## Commands

From the project root (`ShowMyCards/`):
```bash
make dev-frontend     # Run frontend dev server (port 5173)
make build-frontend   # Build SvelteKit frontend
make test-frontend    # Run frontend tests
make types            # Generate TypeScript types from Go models
make lint             # Lint frontend code
make format           # Format frontend code
```

From the frontend directory:
```bash
bun run dev           # Start dev server (default port 5173)
bun run build         # Build for production
bun run preview       # Preview production build (port 4173)
bun run check         # Type-check with svelte-check
bun run format        # Format code with Prettier
bun run lint          # Check code formatting
bun test              # Run all tests (E2E + unit)
bun run test:unit     # Run Vitest component tests
bun run test:e2e      # Run Playwright E2E tests
```

**Note:** Always use `bun` rather than `npm`.

## Configuration

- **Adapter**: `adapter-auto` (auto-detects deployment platform)
- **Preprocessor**: `vitePreprocess()` (TypeScript, PostCSS support)
- **Tailwind**: Vite plugin with forms and typography plugins
- **Vitest**: Browser mode with Playwright provider for component tests

## Type Synchronization with Backend

This frontend receives auto-generated TypeScript types from the Go backend via [tygo](https://github.com/gzuidhof/tygo).

### Generated Files (DO NOT EDIT MANUALLY)

- `src/lib/types/models.ts` - Generated from backend `models/*.go`
- `src/lib/types/api.ts` - Generated from backend `api/*.go`

These types are re-exported from `src/lib/index.ts` for easy importing:

```typescript
import { BACKEND_URL, type Inventory, type SearchResponse } from '$lib';
```

### Regenerating Types

When the backend models change, regenerate TypeScript types:

```bash
# From project root
make types

# Or from frontend directory
bun run types:generate
```

This runs `tygo generate` in the backend directory and updates the generated files.

### After Type Changes

1. **Review the generated type changes** in `src/lib/types/`
2. **Fix TypeScript errors** - The compiler will catch breaking changes
3. **Run type check** - `bun run check` to verify all errors are resolved

### Type Mappings from Backend

Backend Go types are mapped to TypeScript as follows:

| Go Type                          | TypeScript Type     |
| -------------------------------- | ------------------- |
| `string`                         | `string`            |
| `int`, `int64`, `uint`, `uint64` | `number`            |
| `bool`                           | `boolean`           |
| `time.Time`                      | `string` (ISO 8601) |
| `*T` (pointer)                   | `T \| undefined`    |
| `[]T` (slice)                    | `T[]`               |
| `map[K]V`                        | `{ [key: K]: V }`   |

## Working with Svelte/SvelteKit

The Svelte MCP server provides access to comprehensive Svelte 5 and SvelteKit documentation. When working on frontend tasks:

**Available MCP Tools:**

1. **list-sections** - Use FIRST to discover all available documentation sections. Returns a structured list with titles, use_cases, and paths. When asked about Svelte or SvelteKit topics, ALWAYS use this tool at the start to find relevant sections.

2. **get-documentation** - Retrieves full documentation content for specific sections. Accepts single or multiple sections. After calling list-sections, analyze the returned documentation sections (especially the use_cases field) and use get-documentation to fetch ALL relevant sections for the task.

3. **svelte-autofixer** - Analyzes Svelte code and returns issues and suggestions. MUST be used whenever writing Svelte code before sending to the user. Keep calling until no issues or suggestions are returned.

4. **playground-link** - Generates a Svelte Playground link with provided code. Only call after user confirmation and NEVER if code was written to files in their project.

## Working with DaisyUI and UI Libraries

The Context7 MCP server provides up-to-date documentation for any library, including DaisyUI (the Tailwind CSS component library used in this project).

**When to Use Context7:**

- **UI component development** - When implementing forms, modals, cards, tables, or any DaisyUI component
- **Design alignment** - To ensure UI designs align with DaisyUI's built-in functionality and conventions
- **Component customization** - When customizing DaisyUI themes or component styles
- **Third-party libraries** - For documentation on any npm package or framework not covered by specialized MCP servers

**Available MCP Tools:**

1. **resolve-library-id** - MUST be called first to find the Context7-compatible library ID for a package (e.g., "daisyui" → "/saadeghi/daisyui"). Skip this step only if the user provides an explicit library ID in `/org/project` format.

2. **get-library-docs** - Retrieves documentation for a library using the ID from `resolve-library-id`.
   - `mode='code'` (default) - For API references, component usage, and code examples
   - `mode='info'` - For conceptual guides, architecture, and narrative documentation
   - `topic` - Focus on specific topic (e.g., "forms", "modals", "themes")
   - `page` - Pagination for large results (start at 1, try page=2, page=3 if more context needed)

**DaisyUI Usage Pattern:**

```
1. Call resolve-library-id with libraryName="daisyui"
2. Get the Context7 library ID (e.g., "/saadeghi/daisyui")
3. Call get-library-docs with the ID and relevant topic
4. Use mode='code' for component examples and API usage
5. Use mode='info' for theming and customization guides
```

**Best Practices:**

- ALWAYS check DaisyUI documentation before creating custom components
- Use DaisyUI's built-in classes and components instead of writing custom Tailwind
- Verify component props and variants in Context7 docs to avoid unnecessary customization
- For UI tasks, combine Svelte MCP (for reactivity/framework) with Context7 MCP (for styling/components)

## UI Component Library

The application has a set of reusable components in `src/lib/components/`. All components are exported from `src/lib/index.ts`.

### Core Layout Components

**PageHeader** - Consistent page header with title, description, and actions

```svelte
<PageHeader title="Page Title" description="Optional description">
	{#snippet actions()}
		<button class="btn btn-primary">Action</button>
	{/snippet}
</PageHeader>
```

**TableCard** - Card wrapper for tables with optional title and stats

```svelte
<TableCard title="Optional Title">
	{#snippet stats()}
		<StatsCard {stats} />
	{/snippet}

	<table class="table table-zebra">
		<!-- table content -->
	</table>

	{#snippet actions()}
		<button class="btn btn-secondary">Export</button>
	{/snippet}
</TableCard>
```

**StatsCard** - Display statistics in responsive card layout

```svelte
<script>
	const stats = [
		{ title: 'Total', value: 1234, description: 'All time' },
		{ title: 'Active', value: 142, valueClass: 'text-success' }
	];
</script>

<StatsCard {stats} />
```

**EmptyState** - Placeholder when no data exists

```svelte
<EmptyState message="No items found">
	<button class="btn btn-primary">Create First Item</button>
</EmptyState>
```

### Form Components

**FormField** - Standardized form field for modals and forms

- Use for modal forms with vertical stacked layout
- Helper text appears between label and control
- Supports text inputs, selects (via children snippet), and textareas

```svelte
<!-- Text input -->
<FormField
	label="Name"
	id="item-name"
	name="name"
	placeholder="Enter name"
	bind:value={name}
	helper="A descriptive name"
	required />

<!-- Select dropdown -->
<FormField label="Type" id="item-type" required>
	{#snippet children()}
		<select id="item-type" name="type" bind:value={type} class="select select-bordered w-full">
			<option value="a">Option A</option>
			<option value="b">Option B</option>
		</select>
	{/snippet}
</FormField>
```

**SettingRow** - Two-column layout for settings pages

- Use for settings/configuration pages
- Label and description on left, control on right
- Stacks vertically on mobile

```svelte
<div class="divide-y divide-base-300">
	<SettingRow label="Setting Name" description="Description of setting">
		<input type="checkbox" class="toggle toggle-primary" bind:checked={value} />
	</SettingRow>

	<SettingRow label="Another Setting" description="More details">
		<input type="time" class="input input-bordered w-full max-w-xs" bind:value={time} />
	</SettingRow>
</div>

<SettingActions>
	<button class="btn btn-primary">Save Settings</button>
</SettingActions>
```

**SettingActions** - Right-aligned action buttons for settings pages

### UI Components

**Modal** - Dialog component for forms and confirmations

```svelte
<Modal open={showModal} onClose={() => (showModal = false)} title="Modal Title">
	<!-- Modal content -->

	{#snippet actions()}
		<button type="button" class="btn btn-ghost">Cancel</button>
		<button type="submit" class="btn btn-primary">Confirm</button>
	{/snippet}
</Modal>
```

**Important:** For modal forms, action buttons must be inside the `<form>` element. Use `<div class="modal-action">` instead of the Modal's `actions` snippet:

```svelte
<Modal open={showModal} onClose={...} title="Create Item">
  <form method="POST" action="?/create" use:enhance={...}>
    <div class="space-y-4">
      <!-- FormField components -->
    </div>

    <div class="modal-action">
      <button type="button" onclick={() => (showModal = false)} class="btn btn-ghost">Cancel</button>
      <button type="submit" class="btn btn-primary">Create</button>
    </div>
  </form>
</Modal>
```

**Notification** - Alert/toast component for user feedback

```svelte
{#if error}
	<Notification type="error">{error}</Notification>
{/if}
```

**Lozenge** - Badge/tag component for status and labels

```svelte
<Lozenge color="success">Active</Lozenge>
<Lozenge color="error">Failed</Lozenge>
<Lozenge style="outline">Category</Lozenge>
```

**Pagination** - Page navigation for lists

```svelte
<Pagination
	currentPage={data.page}
	totalPages={data.totalPages}
	onPageChange={(page) => goto(`/route?page=${page}`)} />
```

### Card Treatment Utilities

Helper functions for generating card treatment names from Scryfall data:

```typescript
import { getCardTreatmentName, getAvailableTreatments } from '$lib';

// Get treatment name for a specific finish
const treatmentName = getCardTreatmentName(
	card.finishes, // ['nonfoil', 'foil']
	card.frame_effects, // ['showcase', 'surgefoil']
	'foil' // Selected finish
);
// Returns: "Showcase - Surge Foil"

// Get all available treatments for a card
const treatments = getAvailableTreatments(card.finishes, card.frame_effects);
// Returns: [{ key: 'nonfoil', name: 'Showcase' }, { key: 'foil', name: 'Showcase - Surge Foil' }]
```

**Supported finishes:** `nonfoil`, `foil`, `etched`, `glossy`

**Supported special foils:** `surgefoil`, `galaxyfoil`, `oilslickfoil`, `confettifoil`, `halofoil`, `raisedfoil`, `ripplefoil`, `fracturefoil`, `manafoil`, `firstplacefoil`, `dragonscalefoil`, `singularityfoil`, `cosmicfoil`, `chocobofoil`

**Supported style effects:** `inverted`, `showcase`, `extendedart`, `shatteredglass`

## Design Patterns

### Page Structure

All pages should use this consistent structure:

```svelte
<div class="container mx-auto px-4 py-8 max-w-7xl">
	<PageHeader title="Page Title" description="Optional description">
		{#snippet actions()}
			<button class="btn btn-primary">Action</button>
		{/snippet}
	</PageHeader>

	{#if error}
		<Notification type="error">{error}</Notification>
	{/if}

	<!-- Page content -->
</div>
```

### Form Patterns

**Modal Forms** - Use FormField with vertical layout:

- Helper text between label and control
- Bold labels with `required` prop adding `*`
- `space-y-4` for field spacing
- Action buttons inside form with `modal-action` class

**Settings Forms** - Use SettingRow with two-column layout:

- Label/description on left, control on right
- Stacks vertically on mobile
- `divide-y divide-base-300` for dividers
- SettingActions for right-aligned buttons

### Button Hierarchy

1. **Primary** - `btn btn-primary` - Main action (one per section)
2. **Secondary** - `btn btn-secondary` or `btn btn-outline` - Secondary actions
3. **Ghost** - `btn btn-ghost` - Less emphasis, cancel actions
4. **Destructive** - `btn btn-error` - Delete/remove actions

### Icon Usage

- Use Lucide Svelte icons: `import { Icon } from 'lucide-svelte'`
- Icon sizes: `w-4 h-4` for buttons, `w-6 h-6` for decorative
- Always include button title/aria-label for accessibility
- Examples: `Plus`, `Pencil`, `Trash2`, `FolderOpen`, `Box`, `BookOpen`

### Grid Layouts

```svelte
<!-- Responsive card grid -->
<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
	{#each items as item}
		<div class="card bg-base-200 shadow-lg">...</div>
	{/each}
</div>
```

### Responsive Stats

```svelte
<!-- Stacks vertically on mobile, horizontal on desktop -->
<div class="stats stats-vertical lg:stats-horizontal shadow mb-6 w-full">
	<div class="stat">
		<div class="stat-title">Total</div>
		<div class="stat-value">{total}</div>
		<div class="stat-desc">Description</div>
	</div>
</div>
```

## Accessibility Best Practices

- All form fields must have associated labels (use `for` and `id`)
- Required fields marked with `*` via `required` prop
- Error messages associated with controls (FormField `error` prop)
- Icon-only buttons need `title` or `aria-label`
- Keyboard navigation: test tab order, Enter submits forms, Escape closes modals
- Color alone insufficient: use icons + text for status

## Backend Integration

### API Configuration

The backend API URL is configured in `src/lib/config.ts`:

```typescript
export const BACKEND_URL = import.meta.env.VITE_BACKEND_URL || 'http://localhost:3000';
```

Set `VITE_BACKEND_URL` environment variable to override the default.

### Domain Model Reference

For the complete domain model reference, see the backend [CLAUDE.md](../backend/CLAUDE.md#domain-model).

Key models available as TypeScript types:

**Core Models** (from `models.ts`):
- **StorageLocation** - Storage locations for cards (Box, Binder)
- **Inventory** - Cards in collection with ScryfallID, Treatment, Quantity, StorageLocation
- **List** - User-defined card lists (wishlists, decks)
- **ListItem** - Cards in a list with desired/collected quantities
- **SortingRule** - Automated sorting rules with expr-lang expressions
- **Job** - Background job tracking
- **Setting** - Application settings

**API Response Types** (from `api.ts`):
- **SearchResponse** - Paginated search results
- **EnhancedCardResult** - Card with inventory data (this printing, other printings)
- **CardPrices** - Price data (USD, EUR, foil variants)
- **ListSummary** - List with completion statistics
- **EnrichedListItem** - List item with card data (name, set, price)
- **InventoryCardsResponse** - Paginated inventory with card data
- **BatchMoveRequest/Response** - Batch operations
- **ResortResponse** - Re-sorting results with movements
