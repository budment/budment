# Overview

The `blaster parse` command transforms an API specification into a deterministic Endpoint workspace. The lifecycle is deterministic, incremental, and non-destructive.

```mermaid
flowchart LR
    A[API Specification] -->|Parse| B((Endpoint Workspace))
    style A fill:#f9f,stroke:#333,stroke-width:2px
    style B fill:#bbf,stroke:#333,stroke-width:2px
```

---

# Pipeline

The Parse lifecycle consists of seven sequential phases. Each phase has a strictly isolated responsibility. Every phase produces a deterministic workspace for the next phase.

```mermaid
flowchart LR
    P1[Load] --> P2[Synchronize]
    P2 --> P3[Discover]
    P3 --> P4[Resolve]
    P4 --> P5[AI Optional]
    P5 --> P6[Validate]
    P6 --> P7[Write]
```

---

# Phase 1 — Load

Loads the API specification and prepares it for deterministic processing.

## Responsibilities

- Parse the source document (OpenAPI, etc.).
- Normalize and flatten the hierarchical model.
- Deterministic Sort: Sort operations by path depth (shallow before deep) and HTTP method (e.g., GET > POST). This guarantees that canonical endpoints are always processed before derived endpoints.

**Output:** Normalized OpenAPI Model

---

# Phase 2 — Synchronize

Synchronizes existing Endpoint files from the disk with the latest API structure.

## Responsibilities

- Create missing files.
- Remove obsolete generated data.
- Preserve explicit manual fields (e.g., locked targets, ignore).
- Preserve manual files.

**Output:** Working Endpoint (Locked State)

---

# Phase 3 — Discover

Discovers semantic information that can be inferred deterministically, without creating actual relationships. Discovery **MUST NOT** modify manual data.

Examples of discovered entities:

- Identity Candidates (Root / Branch)
- Relative Candidates
- Pending (?)

```mermaid
flowchart TD
    subgraph OpenAPI Schema
        TN[Target Node]
    end

    subgraph Phase 3: Discover
        TN -->|In Request| Rel[Relative Candidate]
        TN -->|In Response| ID[Identity Candidate]

        ID -->|Matches Current API| Root[Root Candidate]
        ID -->|Matches Other API| Branch[Branch Candidate]
        ID -->|Unknown Context| Unbound[Pending Candidate]
    end
```

---

# Phase 4 — Resolve

Resolves the discovered relationships and builds the Canonical Identity Graph. Unresolved items remain Pending (`?`).

## Responsibilities

- Resolve Relatives to Root Identities.
- Resolve Branch Identities to Root Identities.
- Validate Identity graph integrity.
- Detect semantic conflicts.

```mermaid
flowchart LR
    subgraph Phase 4: Resolve
        Branch[Branch Identity] -->|Maps to| Root((Root Identity))
        Relative[Relative Consumer] -->|Maps to| Root
        Pending[Pending / Unbound] -.->|Fails to map| Basket[AI Pending Basket]
    end
    style Root fill:#d4edda,stroke:#28a745,stroke-width:2px
    style Basket fill:#fff3cd,stroke:#ffc107,stroke-width:2px
```

---

# Phase 5 — AI (Optional)

AI may assist in semantic discovery for unresolved (`?`) Target Nodes. AI **NEVER** replaces deterministic results.

AI **MAY**:

- Suggest Identity mappings.
- Suggest Relative mappings.
- Explain ambiguity.

AI **MUST NOT**:

- Overwrite confirmed/locked data.
- Modify deterministic relationships established in Phase 4.

---

# Phase 6 — Validate

Validates the complete Endpoint workspace before writing (Gatekeeping). Validation reports problems but does not modify Endpoint files natively (except downgrading invalid manual references safely back to Pending).

Validation includes detecting:

- Duplicate identities.
- Invalid relatives.
- Orphan branches.
- Invalid or dangling references.
- Consistency checks.

---

# Phase 7 — Write

Persists the Endpoint workspace back to the filesystem. Generated output **MUST** be identical when the input has not changed.

Write **MUST**:

- Preserve user comments natively via AST.
- Preserve formatting and spacing.
- Preserve manual fields.
- Produce deterministic structural ordering.

---

# Lifecycle Rules

| Rule | Description |
|------|-------------|
| Deterministic | Same input → same output. |
| Incremental | Only affected files change. |
| Non-destructive | Manual data (comments, locked targets) is completely preserved. |
| Isolated | Each phase has one single responsibility. |
| Repeatable | Parse can run multiple times safely without corrupting the workspace. |

---

# Error Handling

Errors terminate the current phase immediately (Fail-Fast). Previous completed phases remain unchanged.

```mermaid
flowchart LR
    Load --> Sync --> Discover --> Resolve
    Resolve -- "✖ Error" --> Stop((Stop))

    style Stop fill:#dc3545,color:#fff,stroke:#fff

    Resolve -.-> Validate -.-> Write

    style Validate stroke-dasharray: 5 5,opacity:0.5
    style Write stroke-dasharray: 5 5,opacity:0.5
```

---

Every phase produces a deterministic workspace for the next phase. The final Endpoint workspace becomes the canonical semantic representation of the API.