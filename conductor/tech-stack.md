# VantageData Technology Stack

This document outlines the technologies used in the development of VantageData.

## Core Framework
- **Wails (v2):** Used to build the cross-platform desktop application, bridging the Go backend and the React frontend.

## Backend
- **Go (1.23):** The primary language for backend logic, data processing, and system integration.
- **github.com/getlantern/systray:** Provides system tray functionality for the application.
- **LLM Integration:** Native support for OpenAI, Anthropic, OpenAI-Compatible (Ollama, DeepSeek), and Claude-Compatible APIs.
- **Persistence:** JSON-based local storage for configurations and chat history.

## Frontend
- **React (18):** The primary frontend framework for building the user interface.
- **TypeScript:** Used for type-safe frontend development.
- **Tailwind CSS (3):** The utility-first CSS framework used for styling the application.
- **Vite (Latest):** The build tool and development server for the frontend.
- **Vitest:** Unit testing framework.
- **React Testing Library:** For testing React components.
- **Lucide React:** Icon library for the frontend.
- **React Markdown:** For rendering Markdown in chat.

## Infrastructure & Tools
- **NPM:** Package manager for frontend dependencies.
- **Go Modules:** Package management for the backend.
