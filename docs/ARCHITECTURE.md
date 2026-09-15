---
title: System Architecture
description: Comprehensive technical architecture, node taxonomy, and execution model of the Budment engine.
---

# System Architecture

Budment is architected around a decoupled, compilation-based execution model. Test scenarios defined in TypeScript are translated into an intermediate **Static Execution Graph**, separating high-level workflow definitions from the high-throughput, low-latency requirements of native systems execution.

```mermaid
flowchart TB
    subgraph COMPILATION ["1. COMPILATION LAYER"]
        direction LR
        TS["scenario.ts"] --> ES["esbuild"] --> BN["Bundle"] --> VM["Goja Planning VM"] --> IR["Protobuf Scenario IR"]
    end

    subgraph RUNTIME ["2. RUNTIME ENGINE (Go)"]
        direction TB
        DIR["<b>Director</b><br/><i>(Scenario Orchestrator)</i>"]

        subgraph WORKERS ["Worker Fleet (Concurrent Goroutines)"]
            direction LR
            W1["<b>Worker 1</b><br/>Native FSM"]
            W2["<b>Worker 2</b><br/>Native FSM"]
            WD["..."]
            WN["<b>Worker N</b><br/>Native FSM"]
        end

        subgraph SHARED ["Shared Infrastructure"]
            direction LR
            IO["<b>Shared Transport & I/O Network Engine</b>"]
            POOL["<b>Goja VM Pool (sync.Pool)</b><br/><i>(Borrowed on-demand for JS hooks)</i>"]
        end

        DIR --> W1
        DIR --> W2
        DIR --> WN
        W1 --> IO
        W2 --> IO
        WN --> IO
        WORKERS -.-> |"On-demand hook execution"| POOL
    end

    COMPILATION ==> |"Serialized State Graph"| DIR

    style COMPILATION fill:transparent,stroke:#64748b,stroke-width:1.5px,color:inherit
    style TS fill:transparent,stroke:#94a3b8,color:inherit
    style ES fill:transparent,stroke:#94a3b8,color:inherit
    style BN fill:transparent,stroke:#94a3b8,color:inherit
    style VM fill:transparent,stroke:#60a5fa,color:inherit
    style IR fill:transparent,stroke:#3b82f6,stroke-width:1.5px,color:inherit

    style RUNTIME fill:transparent,stroke:#60a5fa,stroke-width:1.5px,color:inherit
    style DIR fill:transparent,stroke:#60a5fa,stroke-width:1.5px,color:inherit
    style WORKERS fill:transparent,stroke:#93c5fd,stroke-width:1px,color:inherit
    style W1 fill:transparent,stroke:#94a3b8,color:inherit
    style W2 fill:transparent,stroke:#94a3b8,color:inherit
    style WD fill:transparent,stroke:#94a3b8,color:inherit
    style WN fill:transparent,stroke:#94a3b8,color:inherit

    style SHARED fill:transparent,stroke:#94a3b8,stroke-width:1px,color:inherit
    style IO fill:transparent,stroke:#94a3b8,stroke-width:1.5px,color:inherit
    style POOL fill:transparent,stroke:#94a3b8,stroke-dasharray:4 4,stroke-width:1.5px,color:inherit
```

## 1. Two-Phase Execution Lifecycle

### Phase 1: Static Graph Compilation (The Plan Phase)

The engine executes the input bundle within an isolated, side-effect-free compiler VM:

1. **Dependency Packaging:** The scenario definition is bundled in-memory using an embedded compilation pipeline.
2. **DSL Evaluation:** The compiler VM evaluates the bundle against mock builders. Calls to structural definitions (`http.get`, `branch`, `poll`) construct intermediate node descriptors rather than executing network I/O.
3. **Graph Assembly:** The descriptors compile into an immutable Directed Graph (IR) encoded via Protobuf structures, preserving pipeline hierarchy, conditions, and metadata.

### Phase 2: Native FSM Execution (The Run Phase)

Once compilation concludes, the compiler VM is fully decommissioned and the native runtime engine takes control:

1. **Scenario Scheduling:** The `Director` allocates execution groups according to configured order dependencies (`Order`) and timing offsets (`StartAt`).
2. **Worker Allocation:** Virtual Users (VUs) execute as lightweight Go goroutines governed by a Finite State Machine (FSM).
3. **Deterministic State Walking:** Each worker traverses the pre-compiled graph nodes sequentially or conditionally without querying an external script interpreter for routing decisions.

## 2. The Three-Tier Node Taxonomy

Nodes within the Budment architecture are categorized into three distinct functional tiers, governing where and how they are evaluated:

| **1. STRUCTURAL NODES**                                         | **2. OPERATIONAL NODES**                                                          | **3. EXPRESSION NODES**                                                            |
| --------------------------------------------------------------- | --------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------- |
| `Action` (HTTP)                                                 | `Log` / `Abort` / `Fail`                                                          | Context Get                                                                        |
| `Branch` (if/else)                                              | `Script`                                                                          | Random Generators                                                                  |
| `Match` (switch)                                                | `Set` / `Distribute`                                                              | Environment Config                                                                 |
| `Loop` (for/while)                                              | `Metric`                                                                          | Binary / File Buffers                                                              |
| `Poll` (retry)                                                  | `Barrier`/ `Sleep`                                                                | Execution Info                                                                     |
| **Governs topology and routing in the static execution graph.** | **Represents discrete lifecycle actions executed natively by worker goroutines.** | **Resolves dynamic values and expressions without adding structural graph nodes.** |

### Tier 1: Structural Flow Nodes (Graph Topology)

Structural nodes define the branching, iteration, and protocol routing boundaries of a scenario.

- **Nodes:** `ActionNode` (HTTP), `BranchNode`, `MatchNode`, `LoopNode`, `PollNode`.
- **Behavior:** Compiled during Phase 1 into native Go graph structures. During Phase 2, the worker FSM evaluates conditional routing directly in native Go code. JavaScript is only consulted if a condition explicitly requires dynamic hook evaluation.

### Tier 2: Operational Executable Nodes (Lifecycle Actions)

Operational nodes represent discrete instructions executed along a pipeline path.

- **Nodes:** `SleepNode`, `LogNode`, `SetNode`, `DistributeNode`, `MetricNode`, `AbortNode`, `FailNode`, `BarrierNode`...
- **Dual-Context Role:**

  - _In Phase 1:_ They emit lightweight AST descriptor nodes containing configuration parameters (e.g., target duration, metric labels, variable keys).
  - _In Phase 2:_ They trigger immediate native system actions: goroutine timers for sleep, atomic counters for metrics, or cross-worker coordination via `BarrierManager`.

### Tier 3: Expression & Dynamic Data Nodes (Value Providers)

Expression nodes provide dynamic data resolution without adding structural nodes to the AST graph.

- **Nodes:** `get()`, `env()`, `open()`, `random.*`, `info.*`.
- **Template Substitution vs. Hook Evaluation:**

  - _Declarative Templates:_ In static declarations, these nodes compile down to native template tokens (e.g., `{{@env:API_KEY:default}}`, `{{@random:uuid}}`, `{{@open:/path/file:b}}`). At runtime, the native Go template engine interpolates these tokens.
  - _Imperative Hooks:_ Inside JavaScript callbacks, these nodes interact directly with the attached `WorkerScope` via memory bridges, permitting zero-copy reads and writes to worker-local or scenario-shared memory.

## 3. The "Blind SDK" Pattern

The TypeScript SDK contains **no business logic or operational runtime code**. It serves strictly as a type-safe definition layer.

```mermaid
flowchart TD
    TS["<b>TypeScript Scenario</b><br/>Calls SDK function"]
    DISPATCH["<b>globalThis Context Dispatcher</b><br/><i>Resolves active Goja execution environment</i>"]

    subgraph P1 ["PHASE 1: PLANNING"]
        direction TB
        P1_NODE["<b>Goja injected with: Mock Node Builders</b><br/>• Evaluates scenario structure statically<br/>• Returns AST node definitions"]
    end

    subgraph P2 ["PHASE 2: RUNTIME"]
        direction TB
        P2_NODE["<b>Goja injected with: Bridge Functions</b><br/>• Executes operational callbacks and assertions"]
    end

    TS --> DISPATCH
    DISPATCH -->|"During Static Compilation"| P1
    DISPATCH -->|"During Goroutine Execution"| P2

    style TS fill:transparent,stroke:#94a3b8,stroke-width:1.5px,color:inherit
    style DISPATCH fill:transparent,stroke:#94a3b8,stroke-width:1.5px,color:inherit
    style P1 fill:transparent,stroke:#94a3b8,stroke-width:1.5px,color:inherit
    style P2 fill:transparent,stroke:#60a5fa,stroke-width:1.5px,color:inherit
    style P1_NODE fill:transparent,stroke:#94a3b8,stroke-width:1px,color:inherit
    style P2_NODE fill:transparent,stroke:#60a5fa,stroke-width:1px,color:inherit
```

All functions exported by the SDK are direct bindings to `globalThis`. The Go engine binds concrete implementations into the VM based entirely on the active lifecycle phase:

- During Phase 1, `sleep(1)` returns a descriptor object `{ build: () => ({ type: "sleep", duration: 1 }) }`.
- During Phase 2, calling `sleep(1)` within an imperative hook records the sleep interval and yields thread control back to the native Go runtime.

## 4. Inter-Language Yielding: The Panic-Recovery Protocol

Executing blocking operations (such as timers, abort signals, or synchronization barriers) from within synchronous JavaScript callbacks requires transferring control without blocking operating system threads or locking the shared VM pool.

Budment implements a deterministic **Panic-Recovery Control Transfer Protocol**:

```mermaid
sequenceDiagram
    autonumber
    participant GW as Go Worker (Goroutine)
    participant VM as JS VM (sync.Pool)

    GW->>+VM: Attach & Execute hook
    Note over VM: User calls sleep(2) or barrier() inside hook
    VM->>VM: Stage parameters into WorkerContext
    VM->>VM: Trigger panic("BUDMENT_SLEEP")
    Note over VM: Callstack unwinds immediately
    VM-->>-GW: Intercept sentinel panic via recover()

    Note over GW: 1. Release VM back to sync.Pool<br/>2. Execute native time.Sleep() in Go runtime
```

1. **Parameter Staging:** When a blocking operation (`sleep`, `barrier`, `abort`) is invoked inside a JS hook, the Go bridge captures the relevant metadata into the active worker context.
2. **Stack Termination:** The bridge triggers a controlled panic with a known sentinel string (e.g., `BUDMENT_SLEEP`, `BUDMENT_ABORT`). This immediately unwinds the JavaScript execution stack, terminating script execution safely.
3. **Host Interception:** The enclosing Go worker intercepts the sentinel panic via `recover()`, disengages the VM, and returns the instance to the `sync.Pool`.
4. **Native Execution:** The blocking operation is carried out natively in Go using standard synchronization primitives (e.g., `time.NewTimer`, channels, or `sync.Cond`).
