# Mochi CLI Documentation Plan

## Overview
We will generate a Docusaurus-based documentation website for the Mochi CLI tool. The site will feature a playful and welcoming theme, honoring Mochi, the creator's corgi puppy! 🐶

## Docusaurus Setup
- **Framework:** [Docusaurus v3](https://docusaurus.io/)
- **Directory:** `/website` (to separate the site source code from the main Go CLI codebase, though the markdown files will live in `/website/docs`)
- **Theme:** Classic Docusaurus theme with customized corgi-inspired branding (e.g., bone icons, paw prints, or a cute corgi logo).

## Documentation Structure

### 1. Introduction (`docs/intro.md`)
- **Welcome to Mochi:** Introduction to the Multi-Task AI Coding Orchestrator.
- **Why Mochi?** Explanation of parallel worktree execution and automatic PR generation.
- **The Mascot:** A special section dedicated to Mochi the corgi puppy! 🐾

### 2. Getting Started (`docs/getting-started.md`)
- Requirements (Go, Git, Claude/Gemini CLI, GitHub CLI).
- Installation and building from source.
- Quick start commands.

### 3. Usage & Workflows (`docs/usage.md`)
- Task File Format (how to write a `## Tasks` section).
- Examples of workflows:
  - Sprint execution from a local file.
  - Pulling tasks directly from a GitHub Issue.
  - Using the `--sequential` debug mode.
  - Cost-optimized mixed runs (combining Opus, Sonnet, Haiku, Gemini).

### 4. CLI Reference (`docs/cli-reference.md`)
- Exhaustive list of commands (`mochi`, `mochi prune`).
- Explanations of all flags (`--input`, `--issue`, `--model`, `--max-iterations`, `--create-prs`, etc.).

### 5. AI Models (`docs/models.md`)
- Supported models (Claude 3.5 Sonnet, Opus, Haiku; Gemini 1.5/2.0 Pro/Flash).
- Cost vs. Performance recommendations for different task types.

### 6. Architecture & Internals (`docs/architecture.md`)
- How it works: Parser -> Worktree Manager -> Agent Invocation -> GitHub PRs.
- Explanation of the `.mochi_manifest.json` and `.worktrees/` directory.

## Implementation Steps
1. Run `npx create-docusaurus@latest website classic` to bootstrap the site.
2. Update `docusaurus.config.js` with the site title ("Mochi CLI"), tagline ("Multi-Task AI Coding Orchestrator. Also a very good corgi."), and metadata.
3. Replace default Docusaurus docs with our markdown pages based on the `README.md` and `docs/ARCHITECTURE.md`.
4. Add corgi-themed assets (emojis, custom CSS colors) to personalize the landing page (`src/pages/index.js`).
5. Verify the site builds correctly (`npm run build`).
