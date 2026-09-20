---
title: FAQs & Best Practices
description: Core design philosophies, performance optimization, and compile-time safeguards in Budment.
---

# FAQs & Best Practices

## 1. JS Hook Performance

Writing complex logic inside JS hooks is fully supported and optimized. Budment uses a bounded `sync.Pool` of JavaScript VMs (Goja) rather than allocating one VM per Virtual User. This minimizes Garbage Collection (GC) pressure and memory bloat, dedicating maximum CPU cycles to Go's concurrent network I/O.

## 2. DSL Structural Integrity

The Budment DSL is a standard TypeScript array. You cannot physically write imperative statements (like variable assignments or `while` loops) directly inside the array without triggering IDE syntax errors. Logic must be wrapped in `script(() => { ... })`, which strictly isolates runtime execution from the static AST topology.

## 3. Control Flow Analysis (`never` type)

Functions that suspend the JS VM and yield control back to Go (`abort`, `sleep`, `barrier`, `fail`) are strictly typed as `never`. When used inside a runtime hook, your IDE will automatically gray out any subsequent code, providing immediate visual confirmation of the engine's Panic-Recovery mechanism.

```typescript
import { abort, get, script } from "@budment/sdk";

export default [
  script(() => {
    if (get<number>("balance") < 0) {
      abort("Negative balance detected.");
      console.log("Unreachable code - IDE will gray this out.");
    }
  }),
];
```

## 4. Compile-Time Safeguards for AST Proxies

In Phase plan, data providers like `get()` or `env()` return AST Proxy objects, not real values. To prevent silent evaluation bugs during load tests, Budment traps the ECMAScript `Symbol.toPrimitive` lifecycle.

Applying arithmetic operators (`+`, `-`, `*`) or loose comparisons (`==`, `>`, `<`) to a Proxy outside a runtime hook triggers an immediate **Compile Error**.

TypeScript

```typescript
// COMPILE ERROR: Proxy cannot evaluate arithmetic outside a hook.
const nextPage = get('page') + 1;
```

**The Solution:** Use standard Template Literals for static AST string interpolation, and move logical operations into Runtime Hooks where `get()` returns actual data.

TypeScript

```typescript
// VALID: String interpolation builds the static token before entering the pipeline
const targetUrl = `https://api.example.com/users?page=${get("page")}`;
export default [
  http.get(targetUrl).before((req) => {
    // VALID: Math and conditionals safely executed inside a hook
    const page = get<number>("page");
    if (page > 10) {
      req.set({ next_page: page + 1 });
    }
  }),
];
```
