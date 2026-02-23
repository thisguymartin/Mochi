# Product Requirements Document

Sprint tasks for MOCHI to execute. The entire content of this file is sent
as-is to the AI agent as task context -- the bullet format below is for the
agent to interpret, not parsed by MOCHI.

Each task may optionally specify a model using `[model:...]`.
Supported providers are auto-detected from the model name (claude-* or gemini-*).

## Tasks
- Add user auth [model:claude-opus-4-6]
- Fix mobile navbar
- Add dark mode [model:gemini-2.0-flash]
- Write API tests [model:claude-haiku-4-5]
- Migrate users table to PostgreSQL [model:gemini-2.5-pro]
