---
title: Quickstart Guide
description: Step-by-step guide to writing, inspecting, and running your first scenario.
---

# Quickstart Guide

This guide walks you through defining an API scenario, inspecting its compiled execution graph, and executing the workload.

## 1. Define a Scenario Script

Create a script file named `scenario.ts`. In this example, we define an authenticated workflow that evaluates response data, synchronizes workers, and records custom metrics.

```typescript
import { http, branch, poll, barrier, log, metrics, get } from "@budment/sdk";

export const config = {
  vus: 2,
  duration: "10s",
};

export default [
  // 1. Synchronize workers before starting the complex flow
  barrier("sync_start", { quorum: 2 }),
  log("All VUs have passed the synchronization barrier!"),

  http.get("https://httpbin.org/uuid").after(
    { expect: { status: 200 }, extract: { uuid: "request_id" } },
    // Uses standard template literals; resolves natively via Go tokens
    log(`Generated Request ID: ${get("request_id")}`),
  ),

  // 2. Declarative branching based on extracted state
  branch(
    () => !!get("request_id"),
    [
      http
        .post("https://httpbin.org/anything")
        .before({
          headers: { "Content-Type": "application/json" },
          body: { tracking_id: get("request_id") },
        })
        .after({ expect: { status: 200 } }),
    ],
    log("Bypassed downstream request: Missing tracking ID"),
  ),

  // 3. Asynchronous polling (Do-While equivalent)
  poll(
    () => true, // Simulated condition: Loop until this returns true
    [
      http
        .get("https://httpbin.org/status/200")
        .after({ expect: { status: 200 } }),
    ],
    { interval: "1000ms", maxAttempts: 2 },
  ),

  metrics.counter("completed_flows", 1),
];
```

## 2. Inspect the Execution Plan

Before initiating network execution, validate script syntax and review the compiled AST graph using the `plan` command:

```bash
budment plan scenario.ts
```

The CLI outputs an ASCII visualization of your execution plan, verifying node dependencies and metadata without issuing real HTTP requests:

```ast
◆ Default Scenario

▶ SETUP PHASE
--------------------------------------------------------------------------------
└── [HTTP] GET https://httpbin.org/get [id: http_1]

▶ EXECUTION PHASE
--------------------------------------------------------------------------------
├── [HTTP] GET https://httpbin.org/uuid [id: http_2]
│   └── [AFTER] → res_assert, log
├── [BRANCH] Condition: branch_7_cond [id: branch_7]
│   ├── [TRUE]
│   │   └── [HTTP] POST https://httpbin.org/anything [id: http_4]
│   │       ├── [BEFORE] → req_mutate
│   │       └── [AFTER] → res_assert
│   └── [FALSE] → log
└── [POLL] Interval: 1s, Max: 2 [id: poll_10]
    ├── [CHECK] Condition: poll_10_cond
    └── [LOGIC]
        └── [HTTP] GET https://httpbin.org/status/200 [id: http_8]
            └── [AFTER] → res_assert
```

## 3. Execute the Scenario

Run the scenario through the native execution engine:

```bash
budment run scenario.ts
```

You can override runtime parameters on the fly without modifying code:

```bash
budment run scenario.ts --vus 20 --duration 1m
```

During execution, Budment renders a live interactive terminal dashboard (TUI). Once completed, a comprehensive summary report will be generated.

> For advanced CLI options, CI/CD flags, and JSON outputs, refer to the [CLI Command Reference](../reference/cli "null").
