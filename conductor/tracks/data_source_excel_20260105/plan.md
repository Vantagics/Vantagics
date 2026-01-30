# Plan: Excel Data Source Import

## Phase 1: Setup & Dependencies
- [x] **Task 1: Add Go Dependencies.**
    - `go get github.com/xuri/excelize/v2`
    - `go get modernc.org/sqlite`
    - `go mod tidy`

## Phase 2: Backend Core (Data Source Service)
- [x] **Task 2: Create DataSourceService.**
    - Define `DataSource` structs.
    - Implement `LoadDataSources` and `SaveDataSources` (reading/writing `datasources.json`).
- [x] **Task 3: Implement Excel Import Logic.**
    - Implement `ImportExcel(name, filePath)` method.
    - Logic: Open Excel -> Detect Headers/Types -> Create SQLite DB/Table -> Insert Data.
    - Handle empty headers (auto-generate).
- [x] **Task 4: Integrate with App.**
    - Add `DataSourceService` to `App` struct.
    - Expose methods: `GetDataSources`, `AddExcelDataSource`, `DeleteDataSource`.

## Phase 3: Frontend UI
- [x] **Task 5: Create Data Source Components.**
    - `DataSourceList.tsx`: Display list of sources (integrated into Sidebar).
    - `AddDataSourceModal.tsx`: Form for Name, Type (Excel), File.
    - `ContextPanel.tsx`: Data Explorer with table list and preview.
- [x] **Task 6: Integration.**
    - Add "Data Sources" item to Sidebar.
    - Connect UI to Backend methods.
    - Handle "Browse" for file selection using Wails runtime.

## Phase 4: Verification
- [x] **Task 7: Test Import.**
    - Import a sample Excel file (Verified by unit tests).
    - Verify `datasources.json` is updated.
    - Verify SQLite DB is created and contains data.
    - Verify UI updates.
