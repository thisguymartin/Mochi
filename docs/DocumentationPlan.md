# Documentation Plan

Sprint tasks for MOCHI to execute in order to generate Docusaurus documentation for the CLI tool. Add tasks under the `## Tasks` section.
Each task may optionally specify a model using `[model:...]`.
Supported providers are auto-detected from the model name (claude-* or gemini-*).

## Tasks
- Initialize a new Docusaurus project in the `website` directory using the classic theme [model:gemini-2.5-pro]
- Configure `docusaurus.config.js` with site title "Mochi CLI", tagline, and corgi puppy theme branding (e.g., bone favicon, paw print aesthetics) [model:claude-sonnet-4-6]
- Create `website/docs/intro.md` introducing the Mochi CLI (Multi-Task AI Coding Orchestrator) and explaining parallel worktree execution, making sure to show love for Mochi the corgi [model:claude-sonnet-4-6]
- Write `website/docs/getting-started.md` covering Go 1.22+ requirements, git, and model CLI installations [model:claude-haiku-4-5]
- Create `website/docs/usage.md` detailing the task file format and providing workflow examples (e.g., executing from a task file, pulling from GitHub issues) [model:gemini-2.0-flash]
- Write `website/docs/cli-reference.md` detailing all Mochi CLI commands and flags (like `--input`, `--issue`, `--model`, `--create-prs`) [model:claude-sonnet-4-6]
- Create `website/docs/models.md` detailing supported Claude and Gemini models, and their costs/use cases [model:claude-haiku-4-5]
- Create `website/docs/architecture.md` explaining the internals: Parser -> Worktree Manager -> Agent Invocation, based on `READE.md` and `docs/ARCHITECTURE.md` [model:claude-opus-4-6]
- Update the landing page design in `website/src/pages/index.js` to feature a playful and welcoming corgi-themed layout [model:gemini-2.5-pro]
