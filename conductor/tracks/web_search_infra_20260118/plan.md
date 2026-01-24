# Implementation Plan - Web Search and Scraping Infrastructure

## Phase 1: Architecture & Data Model
- [ ] Task: Define Data Structures
    - [ ] Create `SearchEngineConfig` struct (Name, URLPattern, Selectors).
    - [ ] Create `SearchSettings` struct (List of engines, Default engine ID).
    - [ ] Define `WebSearchResult` and `WebPageContent` structs.
- [ ] Task: Configuration Persistence
    - [ ] Write Test: Save and Load SearchSettings to/from `config.json` (or dedicated file).
    - [ ] Implement: Update `app_config.go` (or create new `search_config.go`) to handle persistence.
- [ ] Task: Conductor - User Manual Verification 'Architecture & Data Model' (Protocol in workflow.md)

## Phase 2: Core Service Implementation (Backend)
- [ ] Task: Dependency Management
    - [ ] Run `go get github.com/chromedp/chromedp` and `github.com/PuerkitoBio/goquery`.
- [ ] Task: Chromedp Management Service
    - [ ] Write Test: Initialize and shutdown `chromedp` context.
    - [ ] Implement: `ChromedpService` to manage browser contexts (headless, user-agent, proxy from app config).
- [ ] Task: WebSearch Logic
    - [ ] Write Test: Mock `chromedp` response and test `goquery` parsing logic with defined selectors.
    - [ ] Implement: `ExecuteSearch(query string, engine SearchEngineConfig)` function.
- [ ] Task: WebPageReader Logic
    - [ ] Write Test: Mock HTML content and test text extraction (cleaning scripts/styles).
    - [ ] Implement: `ReadPage(url string)` function using `chromedp` (navigate + wait) and `goquery`.
- [ ] Task: Conductor - User Manual Verification 'Core Service Implementation (Backend)' (Protocol in workflow.md)

## Phase 3: Agent Tool Integration
- [ ] Task: Search Tool Wrapper
    - [ ] Write Test: Verify tool schema and execution mapping to `ExecuteSearch`.
    - [ ] Implement: `WebSearchTool` struct adhering to Agent Tool interface.
- [ ] Task: Reader Tool Wrapper
    - [ ] Write Test: Verify tool schema and execution mapping to `ReadPage`.
    - [ ] Implement: `WebPageReaderTool` struct adhering to Agent Tool interface.
- [ ] Task: Conductor - User Manual Verification 'Agent Tool Integration' (Protocol in workflow.md)

## Phase 4: Frontend Implementation (Settings)
- [ ] Task: Settings UI - Structure
    - [ ] Create `SearchSettings` component in React.
    - [ ] Implement listing of configured search engines.
- [ ] Task: Settings UI - CRUD
    - [ ] Implement "Add/Edit Engine" Modal (inputs for Name, URL, Selectors).
    - [ ] Implement "Set Default" functionality.
- [ ] Task: Backend Binding
    - [ ] Expose `SaveSearchSettings` and `GetSearchSettings` via Wails.
    - [ ] Connect React component to Wails backend.
- [ ] Task: Conductor - User Manual Verification 'Frontend Implementation (Settings)' (Protocol in workflow.md)
