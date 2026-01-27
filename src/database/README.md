# Database Package

This package manages the application database and migrations for VantageData.

## Overview

The database package provides:
- SQLite database initialization
- Migration management system
- Schema versioning and tracking
- Rollback capabilities
- Layout configuration management (LayoutService)
- File management for downloads (FileService)
- Component data availability checking (DataService)

## Services

### LayoutService

Manages dashboard layout configurations with persistence to SQLite database.

**Key Methods:**
- `SaveLayout(config LayoutConfiguration) error` - Saves or updates a layout configuration
- `LoadLayout(userID string) (*LayoutConfiguration, error)` - Retrieves a user's layout
- `GetDefaultLayout() LayoutConfiguration` - Returns the default layout configuration

**Usage:**
```go
layoutService := database.NewLayoutService(db)

// Save a layout
err := layoutService.SaveLayout(config)

// Load a layout
config, err := layoutService.LoadLayout(userID)

// Get default layout
defaultConfig := layoutService.GetDefaultLayout()
```

### FileService

Manages downloadable files with category support (all files and user-request-related files).

**Key Methods:**
- `GetFilesByCategory(category FileCategory) ([]FileInfo, error)` - Retrieves files for a category
- `HasFiles() (bool, error)` - Checks if any files exist in either category
- `DownloadFile(fileID string) (string, error)` - Returns file path for download

**Categories:**
- `AllFiles` - All available files
- `UserRequestRelated` - Files related to user requests

**Usage:**
```go
fileService := database.NewFileService(db, dataDir)

// Get files by category
files, err := fileService.GetFilesByCategory(database.AllFiles)

// Check if any files exist
hasFiles, err := fileService.HasFiles()

// Get file path for download
filePath, err := fileService.DownloadFile(fileID)
```

### DataService

Checks component data availability to support automatic component hiding in the dashboard.

**Key Methods:**
- `CheckComponentHasData(componentType, instanceID string) (bool, error)` - Checks if a component has data
- `BatchCheckHasData(components map[string]string) (map[string]bool, error)` - Checks multiple components efficiently

**Supported Component Types:**
- `metrics` - Key metrics components (checks for data sources)
- `table` - Data table components (checks for data sources)
- `image` - Image display components (checks for image files)
- `insights` - Automatic insights components (checks for insights data)
- `file_download` - File download components (checks FileService.HasFiles())

**Usage:**
```go
fileService := database.NewFileService(db, dataDir)
dataService := database.NewDataService(db, dataDir, fileService)

// Check single component
hasData, err := dataService.CheckComponentHasData("metrics", "metrics-0")

// Batch check multiple components
components := map[string]string{
    "metrics-0": "metrics",
    "table-0": "table",
    "image-0": "image",
}
results, err := dataService.BatchCheckHasData(components)
// results["metrics-0"] -> true/false
```

## Database Location

The database file is created at: `{DataCacheDir}/vantagedata.db`

Where `DataCacheDir` is configured in the application settings.

## Migrations

Migrations are defined in `migrations.go` and are automatically applied when the application starts.

### Current Migrations

1. **Migration 1: Create layout_configs table**
   - Creates the `layout_configs` table for storing dashboard layout configurations
   - Adds index on `user_id` for performance
   - Schema:
     - `id` (TEXT, PRIMARY KEY): Unique identifier for the layout configuration
     - `user_id` (TEXT, NOT NULL, UNIQUE): User identifier (one layout per user)
     - `is_locked` (BOOLEAN, DEFAULT FALSE): Whether the layout is locked
     - `layout_data` (TEXT, NOT NULL): JSON-serialized layout configuration
     - `created_at` (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP): Creation timestamp
     - `updated_at` (TIMESTAMP, DEFAULT CURRENT_TIMESTAMP): Last update timestamp

## Usage

### Initialization

The database is automatically initialized during application startup in `app.go`:

```go
db, err := database.InitDB(dataDir)
if err != nil {
    // Handle error
}
defer db.Close()
```

### Adding New Migrations

To add a new migration:

1. Add a new `Migration` struct to the `GetMigrations()` function in `migrations.go`
2. Increment the version number
3. Provide a description
4. Write the `Up` SQL (to apply the migration)
5. Write the `Down` SQL (to rollback the migration)

Example:

```go
{
    Version:     2,
    Description: "Add new_table",
    Up: `
        CREATE TABLE IF NOT EXISTS new_table (
            id TEXT PRIMARY KEY,
            name TEXT NOT NULL
        );
    `,
    Down: `
        DROP TABLE IF EXISTS new_table;
    `,
},
```

### Rolling Back Migrations

To rollback a specific migration:

```go
err := database.RollbackMigration(db, versionNumber)
```

**Note:** Rollbacks should be used with caution in production environments.

## Testing

Run the test suite:

```bash
cd src/database
go test -v
```

The test suite includes:

**Database & Migrations:**
- Database initialization tests
- Schema validation tests
- Index verification tests
- Migration idempotency tests
- Constraint validation tests
- Default value tests
- Rollback tests

**LayoutService:**
- SaveLayout and LoadLayout tests
- Default layout generation tests
- Transaction handling tests
- Error handling tests

**FileService:**
- File retrieval by category tests
- HasFiles availability checks
- File download path resolution tests
- Empty directory handling tests

**DataService:**
- CheckComponentHasData for all component types (metrics, table, image, insights, file_download)
- BatchCheckHasData for efficient bulk checking
- Error handling for unsupported types
- Performance tests for large batches

## Migration Tracking

The package maintains a `schema_migrations` table that tracks which migrations have been applied:

```sql
CREATE TABLE schema_migrations (
    version INTEGER PRIMARY KEY,
    description TEXT NOT NULL,
    applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

This ensures migrations are only applied once and provides an audit trail.

## Error Handling

- All database operations use transactions to ensure atomicity
- Failed migrations are automatically rolled back
- Detailed error messages include migration version and description
- Database connection failures are logged and reported

## Best Practices

1. **Always test migrations** - Run the test suite before deploying
2. **Write reversible migrations** - Always provide a `Down` migration
3. **Use transactions** - The migration system handles this automatically
4. **Version sequentially** - Use consecutive version numbers
5. **Document changes** - Provide clear descriptions for each migration
