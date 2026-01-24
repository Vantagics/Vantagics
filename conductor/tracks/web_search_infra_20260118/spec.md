# Specification: Web Search and Scraping Infrastructure

## Overview
This track implements the foundational infrastructure required for the agent to access real-time online data. This includes a user-configurable Search Engine management system and two specialized tools for the agent: `WebSearch` and `WebPageReader`. These capabilities will eventually support complex tasks like data comparison and trend prediction.

## Functional Requirements

### 1. Search Engine Management (Settings UI)
- **User Interface:** A new section in the settings to manage search engines.
- **Default Engines:** Pre-configure Google, Baidu, and Bing.
- **Dynamic Configuration:** Users can add/edit search engines with the following fields:
    - `Name`: Display name.
    - `Search URL Pattern`: e.g., `https://www.google.com/search?q={query}`.
    - `Result Container Selector`: CSS selector for individual result items.
    - `Title Selector`: CSS selector relative to the container for the result title.
    - `Link Selector`: CSS selector relative to the container for the result URL.
    - `Snippet Selector`: CSS selector relative to the container for the result summary.
- **Active Selection:** Ability to choose which engine is the "Default" for agent searches.

### 2. Agent Tools
- **WebSearch Tool:**
    - Input: `query` (string).
    - Logic: Executes a search using the active search engine via `chromedp`. Extracts results (Title, URL, Snippet) using `goquery` based on configured selectors.
    - Output: A list of search results.
- **WebPageReader Tool:**
    - Input: `url` (string).
    - Logic: Navigates to the URL using `chromedp`, waits for page load/rendering, and extracts the primary text content using `goquery`.
    - Output: The text content of the page (cleaned of scripts/styles).

### 3. Technical Implementation
- **Browser Automation:** Use `chromedp` to handle JavaScript-heavy search engines and websites.
- **Parsing:** Use `goquery` for efficient HTML element selection and data extraction.
- **Network:** Both tools must respect the application's global proxy settings.

## Non-Functional Requirements
- **Reliability:** Handle timeouts and connection errors gracefully (especially for `chromedp` instances).
- **Performance:** `WebSearch` should ideally return within 10-15 seconds.
- **Resource Management:** Ensure `chromedp` browser contexts are properly closed to avoid memory leaks.

## Acceptance Criteria
- [ ] Users can add a custom search engine and define CSS selectors.
- [ ] The agent can successfully call `WebSearch` and receive structured results from the active engine.
- [ ] The agent can successfully call `WebPageReader` and receive the text content of a specified website.
- [ ] CSS selectors for Google/Baidu/Bing are pre-filled and functional.

## Out of Scope
- The actual "Data Comparison" UI/Dashboard (to be implemented in a future track).
- Advanced anti-bot bypass (e.g., CAPTCHA solving).
