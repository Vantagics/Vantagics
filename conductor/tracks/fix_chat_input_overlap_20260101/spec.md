# Specification: Fix Chat Input and Send Button Layout

## Overview
The chat assistant's input field and send button currently overlap because the button is absolutely positioned inside the input container. We will refactor this to a flexbox layout where the button sits to the right of the input box, ensuring they never overlap and providing a cleaner UI.

## Functional Requirements
- **Refactor Layout in `ChatSidebar.tsx`:**
    - Change the container of the input and button from `relative` to `flex items-center gap-2`.
    - Remove the `absolute` and `right-2.5`, `top-1/2`, `-translate-y-1/2` positioning from the send button.
    - Remove the `pr-16` padding from the input field as it's no longer needed.
    - Ensure the input remains `flex-1` to take up the available width.

## Acceptance Criteria
- [ ] The send button is positioned to the right of the input box.
- [ ] Typing long text in the input box does not go under the send button.
- [ ] Clicking the send button still works as expected.
- [ ] The layout is responsive within the sidebar width.

## Out of Scope
- Changing any backend logic for sending messages.
