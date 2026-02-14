# ShowMyCards Backend

A Go-based REST API server for managing collectible card collections.

This is part of the ShowMyCards monorepo. For development setup, Docker deployment, and build commands, see [DEVELOPMENT.md](../DEVELOPMENT.md) in the project root.

## Technology Stack

- **Framework**: Fiber v3 (FastHTTP-based web framework)
- **Database**: SQLite with GORM ORM
- **Language**: Go 1.25+
- **Type Generation**: Tygo (Go structs → TypeScript)

## Directory Structure

```
ShowMyCards/
├── backend/                     # This directory - Go API server
│   ├── main.go                  # Application entry point, graceful shutdown handling
│   ├── api/                     # HTTP handlers
│   │   ├── bulk_data.go         # Bulk data import operations
│   │   ├── dashboard.go         # Dashboard statistics
│   │   ├── health.go            # Health check endpoint
│   │   ├── inventory.go         # Inventory CRUD + batch operations + resort
│   │   ├── jobs.go              # Background job management
│   │   ├── lists.go             # List CRUD + enriched items with pricing
│   │   ├── scheduler.go         # Job scheduler operations
│   │   ├── search.go            # Scryfall card search with inventory data
│   │   ├── settings.go          # Application settings
│   │   ├── sorting_rules.go     # Sorting rule CRUD + evaluation endpoints
│   │   ├── storage.go           # Storage location CRUD operations
│   │   └── *_test.go            # Test files for each handler
│   ├── database/                # Database layer
│   │   └── client.go            # SQLite connection and lifecycle, migrations
│   ├── models/                  # Domain models (single source of truth)
│   │   ├── base.go              # BaseModel with ID, timestamps
│   │   ├── card.go              # Card data from Scryfall (RawJSON storage)
│   │   ├── inventory.go         # Card inventory (ScryfallID, Treatment, Quantity, StorageLocation)
│   │   ├── job.go               # Background job tracking
│   │   ├── list.go              # User-defined card lists
│   │   ├── list_item.go         # Items within lists
│   │   ├── setting.go           # Application settings
│   │   ├── sorting_rule.go      # SortingRule for automated card sorting
│   │   └── storage.go           # StorageLocation, StorageType enum
│   ├── rules/                   # Rule evaluation engine
│   │   ├── converter.go         # Scryfall card to rule data conversion
│   │   ├── evaluator.go         # expr-lang based rule evaluator
│   │   └── evaluator_test.go    # Rule evaluation tests
│   ├── scryfall/                # Scryfall API client
│   │   └── client.go            # HTTP client for Scryfall API
│   ├── server/                  # Server setup and routing
│   │   ├── server.go            # Fiber app initialization
│   │   ├── routes.go            # Health route registration
│   │   └── *_routes.go          # Feature-specific route registration
│   ├── services/                # Business logic services
│   │   ├── bulk_data.go         # Bulk data import service
│   │   ├── job.go               # Job processing service
│   │   ├── scheduler.go         # Scheduled task management
│   │   └── settings.go          # Settings service
│   ├── utils/                   # Utility functions
│   │   ├── errors.go            # Error handling helpers
│   │   ├── pagination.go        # Pagination utilities
│   │   └── validation.go        # Validation helpers
│   ├── data/                    # SQLite database files (gitignored)
│   └── tygo.yaml                # Type generation config for frontend
├── frontend/                    # SvelteKit web application
│   └── src/lib/types/           # Auto-generated TypeScript types
├── website/                     # Astro marketing site
├── docker/                      # Docker configurations
├── Makefile                     # Build commands
└── DEVELOPMENT.md               # Development setup guide
```

## Key Patterns

- **Dependency injection**: Database client passed to server constructor
- **GORM hooks**: Validation via `BeforeCreate`/`BeforeUpdate` on models
- **Graceful shutdown**: Signal handling for SIGINT/SIGTERM in main.go
- **Single source of truth**: Go structs define the data contract

## Code Reviews

All code reviews must follow BACKEND_REVIEW_STANDARDS.md. Do not flag items listed in the Won't Fix section. Stop the review when only CONSIDER-level findings remain.

## Commands

From the project root (`ShowMyCards/`):

```bash
make dev-backend     # Run backend dev server (port 3000)
make build-backend   # Build Go backend
make test-backend    # Run Go tests
make types           # Generate TypeScript types from Go models
```

From the backend directory:

```bash
go run .             # Run server (default port 3000)
PORT=8080 go run .   # Run on custom port
go test ./...        # Run tests
```

## Type Generation for Frontend

This backend uses [tygo](https://github.com/gzuidhof/tygo) to automatically generate TypeScript types for the frontend.

### Configuration

Type generation is configured in `tygo.yaml`:

```yaml
packages:
  - path: "backend/models"
    output_path: "../frontend/src/lib/types/models.ts"
    type_mappings:
      time.Time: "string"
      uint: "number"
      gorm.DeletedAt: "string | null"
    flatten_embed_structs: true

  - path: "backend/api"
    output_path: "../frontend/src/lib/types/api.ts"
    type_mappings:
      time.Time: "string"
      uint: "number"
      models.Inventory: 'import("./models").Inventory'
    flatten_embed_structs: true
```

### Adding Exported Types

1. **Add the `// tygo:export` comment** above any struct that should be exported to TypeScript:

```go
// tygo:export
type MyNewType struct {
    ID   uint   `json:"id"`
    Name string `json:"name"`
}
```

2. **Use JSON tags** - These determine the TypeScript field names:

```go
type Example struct {
    PublicField  string `json:"public_field"`  // Exported as public_field
    PrivateField string `json:"-"`              // Excluded from TypeScript
}
```

3. **Regenerate types** - Run `make types` from the project root

### Type Mappings

Go types are mapped to TypeScript as follows:

| Go Type                          | TypeScript Type        |
| -------------------------------- | ---------------------- |
| `string`                         | `string`               |
| `int`, `int64`, `uint`, `uint64` | `number`               |
| `bool`                           | `boolean`              |
| `time.Time`                      | `string` (ISO 8601)    |
| `*T` (pointer)                   | `T \| undefined`       |
| `[]T` (slice)                    | `T[]`                  |
| `map[K]V`                        | `{ [key: K]: V }`      |
| `models.X`                       | `import("./models").X` |

### Best Practices

1. **Always add `// tygo:export` comment** to types that should be generated
2. **Use JSON tags** - These determine the TypeScript field names
3. **Regenerate after changes** - Always regenerate types after modifying Go structs
4. **Consistent naming** - Use snake_case in JSON tags for consistency
5. **Cross-package references** - Use type_mappings for types from other packages (e.g., `models.Inventory`)

## API Endpoints

### Health

- `GET /health` - Returns `{"status": "OK"}`

### Dashboard

- `GET /dashboard` - Dashboard statistics (total cards, storage locations, etc.)

### Storage Locations

- `GET /storage` - List storage locations (paginated)
- `GET /storage/:id` - Get single storage location
- `POST /storage` - Create storage location
- `PUT /storage/:id` - Update storage location
- `DELETE /storage/:id` - Delete storage location

### Inventory

- `GET /inventory` - List inventory items (paginated)
  - Query params: `scryfall_id`, `storage_location_id` (or "null" for unassigned)
- `GET /inventory/:id` - Get single inventory item with storage location
- `POST /inventory` - Create inventory item (auto-evaluates sorting rules if no storage location)
- `PUT /inventory/:id` - Update inventory item (partial updates, `clear_storage` flag)
- `DELETE /inventory/:id` - Delete inventory item
- `GET /inventory/cards` - List inventory as enhanced card results with Scryfall data
  - Query params: `page`, `page_size`, `storage_location_id`
- `GET /inventory/by-oracle/:oracle_id` - Get all printings of a card by oracle ID
- `GET /inventory/unassigned/count` - Count inventory items without storage location
- `POST /inventory/batch/move` - Batch move items to a storage location
- `DELETE /inventory/batch` - Batch delete inventory items
- `POST /inventory/resort` - Re-evaluate items against sorting rules

### Lists

- `GET /lists` - List all card lists with summary statistics
- `GET /lists/:id` - Get single list
- `POST /lists` - Create new list
- `PUT /lists/:id` - Update list
- `DELETE /lists/:id` - Delete list (cascade deletes items)
- `GET /lists/:id/items` - List items with enriched card data and value calculations
  - Query params: `page`, `page_size`
- `POST /lists/:id/items` - Batch add items to list
- `PUT /lists/:id/items/:item_id` - Update list item (quantity tracking)
- `DELETE /lists/:id/items/:item_id` - Remove item from list

### Sorting Rules

- `GET /sorting-rules` - List sorting rules (paginated, ordered by priority)
  - Query params: `enabled=true|false` to filter by status
- `GET /sorting-rules/:id` - Get single sorting rule with storage location
- `POST /sorting-rules` - Create sorting rule
- `PUT /sorting-rules/:id` - Update sorting rule (partial updates supported)
- `DELETE /sorting-rules/:id` - Delete sorting rule
- `POST /sorting-rules/evaluate` - Evaluate card data against all enabled rules
- `POST /sorting-rules/validate` - Validate rule expression syntax

### Jobs

- `GET /jobs` - List background jobs (paginated)
  - Query params: `status` (filter by job status)
- `GET /jobs/:id` - Get single job details

### Scheduler

- `GET /scheduler/tasks` - List all scheduled tasks
- `POST /scheduler/tasks` - Create/update scheduled task
- `POST /scheduler/tasks/:name/run` - Manually trigger a task

### Settings

- `GET /settings` - Get application settings
- `PUT /settings` - Update application settings

### Bulk Data

- `POST /bulk-data/import` - Trigger bulk data import from Scryfall

### Card Search

- `GET /search` - Search cards via Scryfall with inventory data
  - Query params: `q` (search query), `page` (default: 1)
  - Returns enhanced results with inventory info (this printing, other printings)
- `GET /search/:id` - Get single card by Scryfall ID

## Domain Model

### BaseModel

All models inherit from BaseModel with:

- `ID` (uint, auto-increment primary key)
- `CreatedAt` (timestamp)
- `UpdatedAt` (timestamp)

### StorageType

Enum with values: `Box`, `Binder`

### StorageLocation

Represents where cards are stored.

- `Name` (string) - Name of the storage location
- `StorageType` (enum: Box, Binder) - Type of storage with database-level validation

### Card

Represents Magic cards from Scryfall's bulk data.

- `ScryfallID` (string, primary key) - Unique Scryfall card identifier
- `OracleID` (string, indexed) - Oracle ID for card versions (can be empty for tokens)
- `RawJSON` (text) - Complete Scryfall card data as JSON (not exposed in API)
- `Name` (string, generated column) - Card name extracted from JSON via SQLite
- `SetCode` (string, generated column) - Set code extracted from JSON via SQLite

**Storage Strategy:**

- Uses SQLite generated columns for frequently queried fields (name, set_code)
- Stores complete Scryfall JSON to avoid duplication and enable flexible queries
- Generated columns are indexed for performance

**Helper Methods:**

- `ToScryfallCard()` - Unmarshals RawJSON to scryfall.Card struct
- `FromScryfallCard()` - Creates Card from scryfall.Card

### Inventory

Represents cards in the collection.

- `ScryfallID` (string, indexed) - Scryfall card identifier for specific printing
- `OracleID` (string, indexed) - Oracle ID for grouping different printings
- `Treatment` (string) - Card treatment/finish (foil, nonfoil, etched, etc.)
- `Quantity` (int) - Number of copies (default: 1, validated >= 0)
- `StorageLocationID` (\*uint, nullable, indexed) - Optional storage location assignment
- `StorageLocation` (relationship) - Preloaded storage location (SET NULL on delete)

**Composite Index:** `idx_oracle_storage` on (oracle_id, storage_location_id) for efficient queries

### List

User-defined card lists (e.g., "Deck - Commander", "Wishlist").

- `Name` (string) - List name
- `Description` (string) - Optional description
- `Items` (relationship) - List items (cards in this list)

### ListItem

Individual cards within a list.

- `ListID` (uint, indexed) - Parent list
- `ScryfallID` (string) - Scryfall card identifier for specific printing
- `OracleID` (string, indexed) - Oracle ID for grouping printings
- `Treatment` (string) - Card treatment/finish
- `DesiredQuantity` (int) - Target number of copies (minimum: 1)
- `CollectedQuantity` (int) - Number of copies currently owned (default: 0)
- `List` (relationship) - Parent list (CASCADE on delete)

**Unique Constraint:** `idx_list_card_treatment` on (list_id, scryfall_id, treatment) prevents duplicates
**Validation:** collected_quantity cannot exceed desired_quantity

### SortingRule

Defines automated rules for sorting cards into storage locations.

- `Name` (string) - Human-readable rule name
- `Priority` (int) - Evaluation order (lower = higher priority)
- `Expression` (string) - expr-lang expression for matching cards
- `StorageLocationID` (uint) - Destination for matching cards
- `Enabled` (bool) - Whether rule is active (default: true)
- `StorageLocation` (relationship) - Preloaded destination location

**Rule Evaluation Engine:**

- Uses `expr-lang/expr` library for expression evaluation
- Rules evaluated sequentially by priority (ascending order)
- First matching rule wins
- Disabled rules are skipped
- Expression syntax: expr-lang (e.g., `prices.usd < 5.0`, `rarity == "mythic"`, `len(colors) > 2`)
- Expressions evaluated against Scryfall card data
- Validation endpoint available to test expressions before saving
- Evaluation endpoint returns matching storage location for given card data

### Job

Background job tracking for long-running operations.

- `Type` (string) - Job type (e.g., "bulk_import", "card_update")
- `Status` (string) - Current status (pending, in_progress, completed, failed)
- `Progress` (int) - Completion percentage (0-100)
- `Error` (string) - Error message if failed
- `StartedAt` (\*time.Time) - When job started
- `CompletedAt` (\*time.Time) - When job finished
- `Metadata` (JSON) - Additional job-specific data

### Setting

Application settings and configuration.

- `Key` (string) - Setting key (unique)
- `Value` (string) - Setting value
- `Description` (string) - What this setting controls

## API Response Types

These types are exported to TypeScript via tygo and used in API responses.

### Search Types (`api/search.go`)

- **SearchResponse** - Paginated search results with `data`, `page`, `total_cards`, `has_more`
- **CardPrices** - Price data (usd, usd_foil, usd_etched, eur, eur_foil, tix)
- **CardResult** - Basic card data from Scryfall
- **CardInventoryData** - Inventory info with `this_printing`, `other_printings`, `total_quantity`
- **EnhancedCardResult** - CardResult + CardInventoryData for search results

### Inventory Types (`api/inventory.go`)

- **InventoryCardsResponse** - Paginated card results with inventory data
- **ExistingPrintingInfo** - Info about a printing in inventory (scryfall_id, treatment, quantity, location)
- **ByOracleResponse** - All printings of a card by oracle ID with unique locations
- **BatchMoveRequest/Response** - Batch move operations
- **BatchDeleteRequest/Response** - Batch delete operations
- **ResortRequest/ResortMovement/ResortResponse** - Re-sorting inventory against rules

### List Types (`api/lists.go`)

- **ListSummary** - List with completion statistics (total items, wanted, collected, percentage)
- **EnrichedListItem** - List item with card data (name, set, rarity, price, finishes)
- **ListItemsResponse** - Paginated items with aggregate stats and value calculations
- **CreateListRequest/UpdateListRequest** - List CRUD operations
- **CreateListItemRequest/UpdateListItemRequest** - List item operations
- **CreateItemsBatchRequest** - Batch add items to list
