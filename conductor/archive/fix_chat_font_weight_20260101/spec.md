# Specification: Non-Bold Font Style in AI Assistant

## Overview
Update the AI Assistant to use a standard (normal) font weight instead of the current medium/bold style in message bubbles and the input field, providing a lighter visual experience.

## Functional Requirements
- **Message Bubbles:** Update `src/frontend/src/components/MessageBubble.tsx` to change the message container from `font-medium` to `font-normal`.
- **Chat Input:** Update `src/frontend/src/components/ChatSidebar.tsx` to change the input field from `font-medium` to `font-normal`.

## Acceptance Criteria
- [ ] User messages in the chat bubbles use normal font weight.
- [ ] AI responses in the chat bubbles use normal font weight.
- [ ] Text entered in the chat input field uses normal font weight.

## Out of Scope
- Changing font styles in the sidebar history or dashboard unless explicitly requested.
