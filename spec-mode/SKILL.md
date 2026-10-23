---
name: spec-mode
description: >
  Carry substantial repository changes end to end, from clarifying requirements
  and defining acceptance criteria through design, planning, implementation,
  verification, independent review, and remediation. Use when work is complex,
  high-impact, spans multiple components or sessions, has meaningful ambiguity
  or quality risk, or needs traceable decisions, progress, and evidence of
  completion. Also use to resume an interrupted Spec Mode workflow. Do not use
  for standalone brainstorming, requirements documentation without subsequent
  implementation, planning, code review, debugging, or simple localized edits;
  use Plan Mode for bounded edits with clear outcomes.
---

# Spec Mode

Use this preference task skill to carry a substantial repository change from
requirements through independently reviewed implementation.

## Scope

Use Spec Mode when:

- the work is complex, high-impact, or spans multiple components or sessions;
- requirements or acceptance expectations need clarification and durable records;
- implementation needs a managed task and remediation queue;
- architecture decisions, technical trade-offs, or API contracts must be captured;
- completion requires traceable evidence and an independent Review gate;
- an interrupted Spec Mode workflow must resume from existing artifacts.

Do not use Spec Mode for simple localized edits or standalone brainstorming,
requirements documentation without subsequent implementation, planning, code
review, debugging, explanation, or repository analysis. For bounded edits with
clear outcomes, use Plan Mode instead.

## Inputs

- The latest user request and its natural language
- The repository root and current codebase state

## Outputs

Create one descriptive folder under `$(cwd)/.doc/` containing:

| Artifact | Purpose | Writable during |
|---|---|---|
| `spec.md` | Requirements and acceptance criteria | Specify; approved requirement changes |
| `design.md` | Technical design, architecture decisions, API/data contracts | Design; approved design changes |
| `tasks.md` | Implementation queue, review issues, status, evidence | Plan and Implement |
| `review.md` | Independent checkpoints, evidence, Review History | Review only |

Write every artifact in the same natural language as the latest user request.

Skipping `design.md` is allowed only when the change is trivial (e.g., a single
bug fix with no architectural decision, no API change, no schema change). When
skipped, create a minimal `design.md` containing only one line:

```markdown
# Design
No additional design decisions — change is trivial and architecture is unchanged.
```

## Delegation Policy

Treat any mechanism that creates a separate work context as a delegation tool.
This includes subagent spawning and subtask delegation through tools such as
`general_purpose_subagent` and `assign_tasks`.

For every delegated task:

- decompose by information boundaries;
- provide the objective, read-first paths, context, scope, success criteria, stop
  condition, and output contract;
- prevent concurrent tasks from writing the same files or shared state;
- independently verify and integrate the returned result.

Apply these phase-specific rules:

| Phase | Delegation rule |
|---|---|
| Specify / Design / Plan | Delegate independent exploration when useful; otherwise continue directly. |
| Implement | When delegation is available and coordination is worthwhile, delegate independent, non-overlapping subtasks concurrently. Keep coupled implementation and tests together; otherwise work serially. |
| Review | When delegation is available, delegate exactly one read-only independent review to a fresh context. Otherwise, run a separate review pass and never treat implementer self-verification as final authority. |

## Workflow

Track these phases in order:

- [ ] Specify
- [ ] Design
- [ ] Plan
- [ ] Approve
- [ ] Implement
- [ ] Review

### 1. Specify

Inspect the codebase and relevant documentation before framing requirements.
Resolve material ambiguity through available user-input tools.

Create `spec.md` with:

- problem, users, goals, and non-goals;
- functional and non-functional requirements;
- constraints, dependencies, assumptions, and open questions;
- Acceptance Criteria typed only as `rule` or `rubric`.

Keep implementation decomposition and technical design out of `spec.md`.

### 2. Design

Translate the approved specification into a technical design. This phase answers
"how to architect the solution" — not "how to build it" (that is Plan).

Create `design.md` with:

- technical approach and architecture overview;
- key design decisions with rationale and alternatives considered (ADR);
- API / interface contracts (endpoints, request/response schemas, method signatures);
- data model or schema changes (tables, fields, indexes, migrations);
- technology or library choices and trade-offs;
- compatibility strategy with existing systems;
- risks and mitigations.

Keep task decomposition, estimates, or execution order out of `design.md`.
That belongs in `tasks.md`.

Read [artifact templates](./references/artifact-templates.md) before creating
`design.md`.

### 3. Plan

Derive `tasks.md` from the completed specification **and** design:

- map every Acceptance Criterion to implementation work;
- use `design.md` decisions as the decomposition basis;
- create atomic, dependency-ordered vertical slices;
- assign `high`, `medium`, or `low` priority;
- derive task-local Test Requirements typed only as `rule` or `rubric`;
- preserve the parent criterion type unless a narrower rule supplies evidence for
  a rubric.

Do not create `review.md` during Specify, Design, or Plan.

Read [artifact templates](./references/artifact-templates.md) before creating the
first `spec.md` or `tasks.md`.

### 4. Approve

Validate `spec.md`, `design.md`, and `tasks.md`, then notify or ask the user to
review all three artifacts using any available interaction mechanism. Wait for
explicit approval before implementation.

If approval changes requirements, return to Specify; if it changes architecture
decisions, return to Design; then regenerate the affected downstream artifacts
before requesting approval again.

### 5. Implement

Process one ready item at a time:

1. Resume `in_progress` work; otherwise choose the highest-priority ready
   `pending` item.
2. Set `Status: in_progress`.
3. Apply the Implement delegation policy.
4. Self-verify every `rule` TR and every `rubric` TR. Record rubric score,
   rationale, and evidence.
5. Iterate until every task-local TR passes.
6. Add `Completion Evidence`, then set `Status: completed`.
7. Continue until the queue drains.

During Implement, treat `review.md` as read-only even when a prior Review cycle
created it. Treat `design.md` as the authoritative architecture reference; do
not modify it during implementation — if design changes are required, return to
Design and re-approve.

A queue is drained only when:

```text
all tasks/issues IN {completed, cancelled}
AND none IN {pending, in_progress, blocked}
AND every cancelled item has explicit user approval
AND required acceptance coverage remains intact
```

If blocked work prevents progress, persist `Blocked By` and `Unblock Condition`
and request the required resolution. A blocked queue is not drained.

### 6. Review

Enter Review only after the queue drains.

1. Reconcile all current ACs/TRs against `review.md`.
2. Create `review.md` if absent; otherwise preserve history and add missing
   checkpoints for remediation or approved requirement changes.
3. Apply the Review delegation policy and give the reviewer the independent
   review contract. A reviewer-specific agent type or tool is not required.
4. Route the structured result:
   - `pass`: update `review.md` and finish;
   - `fail`: update `review.md`, return to Implement, then materialize every
     actionable finding as a pending issue before selecting work;
   - `blocked`: record the blocker in Review History and request resolution.

After remediation drains the queue, start a new Review cycle with a fresh
reviewer. After a blocked Review is unblocked, also start a new Review cycle with
a fresh reviewer.

Give every fresh reviewer:

- the user goal and repository root;
- absolute paths to `spec.md`, `design.md`, `tasks.md`, and `review.md`;
- relevant run instructions and environment constraints;
- implementation artifacts and task Completion Evidence.

Read [artifact templates](./references/artifact-templates.md) before generating
`review.md`, constructing the reviewer prompt, or creating review issues.

## Verification Vocabulary

| Type | Use | Required shape |
|---|---|---|
| `rule` | Objectively verifiable binary condition | Observable pass condition and evidence source |
| `rubric` | Evaluative quality dimension | Dimension, numeric scale, low/mid/high anchors, pass threshold, evidence source |

Every AC/TR has exactly one type: `rule` or `rubric`. Implementers self-verify
task-local rules and rubrics and record evidence. Independent Review separately
re-verifies both types and remains the final acceptance gate.

## Item Status

| Status | Meaning | Required side effect |
|---|---|---|
| `pending` | Not started | None |
| `in_progress` | Being implemented or repaired | Remove stale blocker fields |
| `blocked` | Cannot proceed autonomously | Add `Blocked By` and `Unblock Condition` |
| `completed` | All task-local rule and rubric TRs pass self-verification | Add `Completion Evidence`, including rubric score, rationale, and evidence |
| `cancelled` | User-approved removal from scope | Add reason and approval evidence; preserve dependencies and AC coverage |

Use stable headings such as `## Task 1: ...` and store state only in the `Status`
field. Local verification failure remains `in_progress`; do not add a `failed`
status.

## Review Result Contract

| Result | Required condition | Route |
|---|---|---|
| `pass` | Every required checkpoint passes, every AC has independent evidence, and no actionable finding or blocked check remains | Finish |
| `fail` | At least one actionable finding exists and every failed checkpoint maps to one | Return to Implement, then create a non-empty pending remediation queue before selecting work |
| `blocked` | A required check cannot run because of environment, permission, or dependency | Record blocker and request resolution |

Use `blocked`, not `fail`, when unavailable evidence is the only problem.
Advisory findings do not block acceptance.

## Completion

Finish only when:

```text
all tasks/issues IN {completed, user-approved cancelled}
AND all required review checkpoints are checked
AND every rule has passing evidence
AND every rubric meets its threshold with rationale and evidence
AND latest Review result == pass
AND no actionable finding remains
```

## Validation for the Spec Mode Workflow

Run rules before rubrics.

### Rules

- `spec.md`, `design.md`, and `tasks.md` precede approval and implementation.
- `review.md` is created or changed only during Review.
- `design.md` is not modified during Implement; design changes return to the Design phase.
- AC/TR types are exclusively `rule` or `rubric`.
- Task headings contain no status markers.
- Every completed item has `Completion Evidence`.
- Every failed review creates at least one pending remediation issue.
- A passing Review is the only success exit.

### Rubrics

- **Workflow fidelity (0-2)**:
  - `2`: all six phases and artifact boundaries are followed exactly;
  - `1`: the workflow succeeds with one non-critical boundary mistake;
  - `0`: phases are skipped or review independence is broken.
- **Adaptability (0-2)**:
  - `2`: task decomposition and evidence choices fit the repository;
  - `1`: usable but overly rigid or generic;
  - `0`: workflow cannot adapt to the request.

## Gotchas

- Do not collapse Specify, Design, and Plan: requirements define what; design
  defines architecture; tasks define execution.
- Do not stuff design decisions into `spec.md` — design goes in `design.md`.
- Do not stuff task decomposition into `design.md` — that belongs in `tasks.md`.
- Do not treat implementer self-verification as independent Review evidence.
- Review records findings in `review.md`; Implement materializes actionable
  findings in `tasks.md` before selecting work.
- `design.md` can be a one-liner for trivial changes, but never skip the file
  entirely — the explicit "no design needed" record is itself valuable context.
