# AGENTS.md

Purpose
- This file orients code-generating agents (Codex, Claude Code, etc.) to the repository: where to find the plan and spec, what each upstream doc is for, what lives in /designs/, and the operational rules agents must follow while making changes.

Quick start
- Read prompt_plan.md first — it is the agent-driven plan that contains step-by-step prompts, checklists, and tests.
- Consult spec.md for minimal functional and technical requirements and the Definition of Done.
- Use idea.md and idea_one_pager.md for product/context-level background.
- Use /designs/ for visual assets and design specifications.
- Always follow the agent responsibilities and guardrails in the repository docs section below.

Repository file descriptions
- prompt_plan.md
  - Agent-Ready Planner. Contains the staged plan, per-step prompts (copy-pasteable to LLMs), expected artifacts, tests, rollback/idempotency notes, and a TODO checklist using Markdown checkboxes.
  - This file is the authoritative driver of the automated/code-generation workflow. Agents must update its checkboxes as they complete work (see Repository docs below).

- spec.md
  - Minimal functional and technical specification. Contains the expected behavior, constraints, APIs, data layouts, and a concise Definition of Done for the current milestone/MVP.
  - Agents must not introduce new public APIs without updating spec.md and related tests.

- idea.md
  - Informal notes and brainstorming about the product, tradeoffs, longer-term ideas, and rationale. Useful for context but not authoritative for implementation decisions.

- idea_one_pager.md (or idea_one_pager)
  - Short, high-level summary (problem, audience, core flow, and primary value proposition). Use this for quick product-context orientation.

What lives in /designs/
- The /designs/ directory holds design artifacts that guide the UI/UX and visual decisions. Typical contents:
  - Wireframes and mockups (SVG, PNG, sometimes PDF)
  - Figma export links or an exported snapshots folder (PNG/SVG)
  - Icon and font licensing information
  - Design tokens, color palettes, and spacing rules
  - Component sketches and state diagrams
  - Readme.md explaining the provenance and how to open/edit the files
- Naming and size guidelines:
  - Keep files small and source-control friendly (prefer SVG/PNG optimized; do not store huge layered PSD/Figma source files unless necessary).
  - If a large binary file is required, document why and where the canonical source is (e.g., Figma link). Prefer storing lightweight exports in /designs/ and leaving master files in an external design tool.
  - Include a short "open with" note and license attribution where applicable.

Include the following repository docs section verbatim (agents must follow this exactly):

## Repository docs
- 'ONE_PAGER.md' - Captures Problem, Audience, Platform, Core Flow, MVP Features; Non-Goals optional.
- 'DEV_SPEC.md' - Minimal functional and technical specification consistent with prior docs, including a concise **Definition of Done**.
- 'PROMPT_PLAN.md' - Agent-Ready Planner with per-step prompts, expected artifacts, tests, rollback notes, idempotency notes, and a TODO checklist using Markdown checkboxes. This file drives the agent workflow.
- 'AGENTS.md' - This file. 

### Agent responsibility
- After completing any coding, refactor, or test step, **immediately update the corresponding TODO checklist item in 'prompt_plan.md'**.  
- Use the same Markdown checkbox format ('- [x]') to mark completion.  
- When creating new tasks or subtasks, add them directly under the appropriate section anchor in 'prompt_plan.md'.  
- Always commit changes to 'prompt_plan.md' alongside the code and tests that fulfill them.  
- Do not consider work “done” until the matching checklist item is checked and all related tests are green.
- When a stage (plan step) is complete with green tests, update the README “Release notes” section with any user-facing impact (or explicitly state “No user-facing changes” if applicable).
- Even when automated coverage exists, always suggest a feasible manual test path so the human can exercise the feature end-to-end.
- After a plan step is finished, document its completion state with a short checklist. Include: step name & number, test results, 'prompt_plan.md' status, manual checks performed (mark as complete only after the human confirms they ran to their satisfaction), release notes status, and an inline commit summary string the human can copy & paste.

#### Guardrails for agents
- Make the smallest change that passes tests and improves the code.
- Do not introduce new public APIs without updating 'spec.md' and relevant tests.
- Do not duplicate templates or files to work around issues. Fix the original.
- If a file cannot be opened or content is missing, say so explicitly and stop. Do not guess.
- Respect privacy and logging policy: do not log secrets, prompts, completions, or PII.

#### Deferred-work notation
- When a task is intentionally paused, keep its checkbox unchecked and prepend '(Deferred)' to the TODO label in 'prompt_plan.md', followed by a short reason.  
- Apply the same '(Deferred)' tag to every downstream checklist item that depends on the paused work.
- Remove the tag only after the work resumes; this keeps the outstanding scope visible without implying completion.




#### When the prompt plan is fully satisfied
- Once every Definition of Done task in 'prompt_plan.md' is either checked off or explicitly marked '(Deferred)', the plan is considered **complete**.  
- After that point, you no longer need to update prompt-plan TODOs or reference 'prompt_plan.md', 'spec.md', 'idea_one_pager.md', or other upstream docs to justify changes.  
- All other guardrails, testing requirements, and agent responsibilities in this file continue to apply unchanged.


---

## Testing policy (non‑negotiable)
- Tests **MUST** cover the functionality being implemented.
- **NEVER** ignore the output of the system or the tests - logs and messages often contain **CRITICAL** information.
- **TEST OUTPUT MUST BE PRISTINE TO PASS.**
- If logs are **supposed** to contain errors, capture and test it.
- **NO EXCEPTIONS POLICY:** Under no circumstances should you mark any test type as "not applicable". Every project, regardless of size or complexity, **MUST** have unit tests, integration tests, **AND** end‑to‑end tests. If you believe a test type doesn't apply, you need the human to say exactly **"I AUTHORIZE YOU TO SKIP WRITING TESTS THIS TIME"**.

### TDD (how we work)
- Write tests **before** implementation.
- Only write enough code to make the failing test pass.
- Refactor continuously while keeping tests green.

**TDD cycle**
1. Write a failing test that defines a desired function or improvement.  
2. Run the test to confirm it fails as expected.  
3. Write minimal code to make the test pass.  
4. Run the test to confirm success.  
5. Refactor while keeping tests green.  
6. Repeat for each new feature or bugfix.

---

## Important checks
- **NEVER** disable functionality to hide a failure. Fix root cause.  
- **NEVER** create duplicate templates or files. Fix the original.  
- **NEVER** claim something is “working” when any functionality is disabled or broken.  
- If you can’t open a file or access something requested, say so. Do not assume contents.  
- **ALWAYS** identify and fix the root cause of template or compilation errors.  
- If git is initialized, ensure a '.gitignore' exists and contains at least:
  
  .env
  .env.local
  .env.*
  
  Ask the human whether additional patterns should be added, and suggest any that you think are important given the project. 

## When to ask for human input
Ask the human if any of the following is true:
- A test type appears “not applicable”. Use the exact phrase request: **"I AUTHORIZE YOU TO SKIP WRITING TESTS THIS TIME"**.  
- Required anchors conflict or are missing from upstream docs.  
- You need new environment variables or secrets.  
- An external dependency or major architectural change is required.
- Design files are missing, unsupported or oversized

[End verbatim section]

Operational notes (short)
- Always commit prompt_plan.md updates with code and tests in the same commit.
- Keep each commit small and focused; ensure go/build/tests (or equivalent) succeed locally before proposing changes.
- If a requested file is missing or can't be opened, stop and report the problem — do not guess.
- Suggest a manual test plan for each code change even when automated tests exist.

If you need anything changed in this AGENTS.md (formatting, more examples, or stricter rules), say what you'd like and I will update it.

<!-- Generated with vibescaffold.dev -->
