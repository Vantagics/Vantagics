# VantageData Database & Storage Services

This package contains services for managing application state, dashboard layouts, and file operations.

## Services

### LayoutService (`layout_service.go`)
Manages dashboard drag-and-drop layout configurations.
- **Storage**: JSON file (`layout_configs.json` in the data directory).
- **Features**: Supports multiple user profiles, automatic ID generation, and timestamping.

### DataService (`data_service.go`)
Provides methods for checking component data availability.
- **Features**: Batch checking of data for dashboard components (metrics, tables, images, etc.).
- **Integration**: Works with `DataSourceService` to verify if data sources are populated.

### FileService (`file_service.go`)
Manages downloadable files generated during analysis.
- **Storage**: Local filesystem directories (`files/` and `user_requests/`).
- **Features**: Category-based file retrieval, secure download path resolution with path traversal protection.

### ExportService (`export_service.go`)
Coordinates dashboard data export operations.
- **Formats**: Supports JSON, XLSX (placeholder), and CSV.
- **Features**: Automatic filtering of empty components, data collection, and export result tracking.

## Implementation Details

The application has moved from a traditional SQL database (SQLite/DuckDB) for configuration storage to a more lightweight **JSON-based storage** model. This simplifies deployment and improves reliability for desktop usage.

Analysis-specific data (cached from Excel/CSV) is still handled by the `agent` package using **DuckDB** for high-performance OLAP queries.
