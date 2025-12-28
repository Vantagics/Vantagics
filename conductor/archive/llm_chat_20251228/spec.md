# Track Specification: LLM-Based Chat Interface

## Overview
This track involves adding a conversational interface to RapidBI, allowing non-technical users to analyze their data and interact with the application using natural language.

## Goals
- Create a collapsible sidebar/drawer for the chat interface.
- Implement backend logic to communicate with OpenAI and Anthropic API providers.
- Enable the presentation of rich text, integrated visualizations, and suggested actions within the chat.

## Functional Requirements
- **Sidebar Integration:** A toggleable sidebar that preserves the user's primary view (e.g., dashboard).
- **Multi-Provider Support:** Backend configuration and logic to switch between OpenAI and Anthropic.
- **Data Analysis Interface:** Users can input queries about their data.
- **Rich Responses:** Display LLM outputs using Markdown, including tables.
- **Visual Insights:** Ability to render charts or metric widgets directly in the chat history based on LLM suggestions.
- **Contextual Actions:** Buttons for subsequent actions (e.g., "Export to PDF") provided as part of the response.

## Technical Requirements
- **Frontend:** React, TypeScript, Tailwind CSS, Lucide React icons.
- **Backend:** Go (Wails) for API integration and data processing.
- **TDD:** Frontend and backend tests ensuring message flow and API handling.
- **API Keys:** Secure handling of user-provided API keys (stored via Config).

## Acceptance Criteria
- [ ] User can open and close the chat sidebar.
- [ ] User can choose between OpenAI and Anthropic providers in settings.
- [ ] Chat messages are sent to the selected provider and responses are displayed.
- [ ] Markdown formatting is correctly rendered in chat bubbles.
- [ ] At least one mock visualization is rendered in chat when requested.
