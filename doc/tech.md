## tech stack

go + wails 2.0 + React + TypeScript + Tailwind CSS

All source code is located in src dir.

## modules

- Data source management (SQL, NoSQL DB, and other data sources), including local data cache management
- LLM Model Configuration (support Claude, OpenAI, Qwen, GLM, Deepseek, MiniMax, Gemini)
- LLM based Agent management, by Business Domain and Common Domain
- Reports/Results, display the analysis results
- Quick Analysis Pack system for replayable analysis workflows
- Marketplace integration for sharing and downloading analysis packs
- License server integration for SN + Email authentication

## appearance 

Modern UI, slightly blue.

- The left side is the data source management, by the name of data source
- The bottom of the main UI is the chat box
- The right side is the context sensitive UI
  - Display the data grid for results or Reports

## plugin system

- Support Anthropic skills
- Support LLM MCP protocol

## marketplace system

- SN + Email based authentication (integrated with License server)
- Four pricing models: Free, Per Use, Time Limited, Subscription
- Local usage license management for paid packs
- Credits system for purchasing paid packs
- Category-based browsing (Shopify, BigCommerce, eBay, Etsy, and custom categories)
     