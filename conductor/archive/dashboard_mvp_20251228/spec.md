# Track Specification: Smart Dashboard MVP

## Overview
This track aims to deliver the initial "Smart Dashboard" for RapidBI. It focuses on presenting a clean, approachable interface for non-technical users to view business data summaries and automated insights.

## Goals
- Establish a foundational dashboard UI using React and Tailwind CSS.
- Implement a mock data service to simulate business data retrieval.
- Create a "Smart Insight" component that displays natural language summaries of the data.
- Adhere to the "Friendly and Approachable" and "Visual Richness" product guidelines.

## Functional Requirements
- **Data Summary View:** Display key metrics (e.g., Total Sales, Active Users) in an intuitive layout.
- **Natural Language Insights:** Display at least two automated insights generated from the mock data (e.g., "Sales increased by 15% this week!").
- **Responsive Design:** Ensure the dashboard is usable on different screen sizes.

## Technical Requirements
- **Frontend:** React, TypeScript, Tailwind CSS.
- **Backend:** Go (Wails) for data provision (mocked initially).
- **TDD:** All features must have corresponding unit tests with >80% coverage.
