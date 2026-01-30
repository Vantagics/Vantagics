# Specification: Excel Data Source Import

## Overview
Enable users to import Excel files as data sources. Data will be stored in a local SQLite database within the configured Data Cache Directory.

## Functional Requirements
1.  **Data Source Management**:
    *   Maintain a registry of data sources in `datasources.json` located in the `DataCacheDir`.
    *   Each entry should track: `id` (UUID), `name`, `type` (e.g., "excel"), `created_at`, and path to the storage (relative to `DataCacheDir`).

2.  **Excel Import**:
    *   User inputs: `Name`, `Driver Type` (select "Excel"), and `File` (via file picker).
    *   System reads the Excel file (first sheet by default or all? Prompt implies "import data", usually implies the active sheet or all. I'll assume first sheet for MVP).
    *   **Schema Inference**:
        *   Read first row as headers.
        *   If a header is empty, generate a name like `field_<index>_<type>`.
        *   Infer column types (Text, Integer, Real) based on data in the second row.
    *   **Storage**:
        *   Create a directory `sources/<uuid>/`.
        *   Create SQLite DB `sources/<uuid>/data.db`.
        *   Create table (name = "import_data" or sheet name).
        *   Bulk insert rows.

3.  **UI/UX**:
    *   New "Data Sources" section in the application.
    *   List of existing sources.
    *   "Add Data Source" modal/form.
    *   Feedback on success/failure.

## Technical Constraints
*   Use `github.com/xuri/excelize/v2` for Excel processing.
*   Use `modernc.org/sqlite` for SQLite (pure Go, no CGO).
*   Data stored in `Config.DataCacheDir`.

## Data Structure (`datasources.json`)
```json
[
  {
    "id": "uuid-string",
    "name": "Sales Data Q1",
    "type": "excel",
    "created_at": "2026-01-05T...",
    "config": {
      "original_file": "/path/to/file.xlsx",
      "db_path": "sources/<uuid>/data.db",
      "table_name": "Sheet1"
    }
  }
]
```
