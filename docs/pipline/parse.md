# Overview

The `blaster parse` command transforms an API specification into a deterministic Endpoint workspace. The lifecycle is deterministic, incremental, and non-destructive.

```mermaid
flowchart LR
    A[API Specification] -->|Parse| B((Endpoint Workspace))

    style A fill:#2d3748,stroke:#60a5fa,stroke-width:2px,color:#ffffff
    style B fill:#1e3a8a,stroke:#60a5fa,stroke-width:2px,color:#ffffff
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

- Parse the source document via Protocol Adapters (e.g., OpenAPI, gRPC proto).
- Normalize and flatten the hierarchical structure into a Unified `schema.Model`.
- Deterministic Sort: Sort operations by path depth (shallow before deep) and method priority. This guarantees that canonical endpoints are always processed before derived endpoints.

**Output:** Unified Schema Model

---

# Phase 2 — Synchronize

Synchronizes existing Endpoint files from the disk with the latest API structure.

## Responsibilities

- Create missing files.
- Remove obsolete generated data (Garbage Collection).
- Preserve explicit manual fields (e.g., locked targets, ignore).
- Isolate synchronization by protocol namespace (e.g., OpenAPI Sync only affects the `rest/` directory).

**Output:** Working Endpoint (Locked State)

---

# Phase 3 — Discover

Discovers semantic information that can be inferred deterministically, without creating actual relationships. Discovery **MUST NOT** modify manual data.

Examples of discovered entities:

- Identity Candidates (Root / Branch)
- Relative Candidates
- Pending (`?`)

```mermaid
flowchart TD
    subgraph Unified Schema Model
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

- Learn from existing manual mappings (Learned Dictionary) for transitive resolution.
- Resolve Relatives to Root Identities.
- Resolve Branch Identities to Root Identities.
- Build the protocol-agnostic directed graph.

```mermaid
flowchart LR
    subgraph Phase 4: Resolve
        Branch[Branch Identity] -->|Maps to| Root((Root Identity))
        Relative[Relative Consumer] -->|Maps to| Root
        Pending[Pending / Unbound] -.->|Fails to map| Basket[AI Pending Basket]
    end

    style Root fill:#14532d,stroke:#22c55e,stroke-width:2px,color:#ffffff
    style Basket fill:#78350f,stroke:#f59e0b,stroke-width:2px,color:#ffffff
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

# Phase 6 — Validate (Gatekeeper)

Validates the complete Endpoint workspace before writing. Validation reports problems but does not modify Endpoint files natively (except downgrading invalid manual references safely back to Pending).

## Responsibilities

- Detect duplicate identities.
- Detect invalid relatives.
- Detect orphan branches.
- Intercept invalid or dangling references.
- Perform cross-protocol consistency checks.

---

# Phase 7 — Write

Persists the Endpoint workspace back to the filesystem. Generated output **MUST** be identical when the input has not changed.

Write **MUST**:

- Preserve user comments natively via AST.
- Preserve formatting and spacing.
- Produce deterministic structural ordering.
- Act as a SerDes Boundary: Automatically strip redundant protocol prefixes for intra-protocol addresses to ensure an optimal Developer Experience (DX).

---

# Lifecycle Rules

| Rule | Description |
|------|-------------|
| Deterministic | Same input → same output. |
| Config-Driven | Behavior is controlled strictly by the project's YAML configuration. |
| Incremental | Only affected files change. |
| Non-destructive | Manual data (comments, locked targets) is completely preserved. |
| Isolated | Each phase has one single responsibility. |
| Repeatable | Parse can run multiple times safely without corrupting the workspace. |

---

# Error Handling

Errors terminate the current phase immediately (Fail-Fast). Previous completed phases remain unchanged. Syntax errors in existing YAML files trigger a fail-safe mechanism, returning the file untouched to prevent data loss.

```mermaid
flowchart LR
    Load --> Sync --> Discover --> Resolve
    Resolve -- "✖ Error" --> Stop((Stop))

    Resolve -.-> Validate -.-> Write

    style Stop fill:#7f1d1d,stroke:#ef4444,stroke-width:2px,color:#ffffff
    style Validate stroke-dasharray:5 5,opacity:0.6
    style Write stroke-dasharray:5 5,opacity:0.6
```

---

Every phase produces a deterministic workspace for the next phase. The final Endpoint workspace becomes the canonical, protocol-agnostic semantic representation of the API.

---