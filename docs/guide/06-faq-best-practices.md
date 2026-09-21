---
title: FAQs & Anti-Patterns
description: Core design philosophies, performance optimization, and compile-time safeguards in Budment.
---

# FAQs, Anti-Patterns & Best Practices

## 1. The "God Hook" Myth & Concurrency

**The Concern:** _If I put too much logic into JS hooks, won't it create a bottleneck, block the engine, and defeat the purpose of Go's high concurrency?_

**The Reality:** Writing robust logic inside JS hooks is not a "dirty" anti-pattern—it is exactly how the engine is designed to be used safely and efficiently. Borrowing a VM from the pool is highly optimized for performance:

- **Hardware Reality Check:** Concurrency is not parallelism. Even if you spawn 10,000 Virtual Users (goroutines), a 4-core CPU can only physically execute 4 instructions simultaneously. The Go scheduler multiplexes the rest.
- **Reducing Heap & GC Pressure:** JavaScript runtimes carry heavy internal states (Global objects, prototype chains). If Budment spawned 10,000 isolated JS VMs for 10,000 VUs, the Go Garbage Collector (GC) would drown trying to traverse a massive object graph, causing severe CPU spikes. By using a bounded pool, Budment keeps the memory footprint minimal, dedicating full CPU power to network I/O.

## 2. You Can't "Break" the DSL

**The Concern:** _What if someone writes standard JavaScript logic (like `let a = 1`, or `while` loops) directly in the DSL array instead of a hook?_

**The Reality:** The Budment DSL is protected by standard TypeScript grammar.
Because your scenario is structurally just an array of object instances (`[ node1, node2 ]`), you physically cannot write imperative statements like `const x = 10` directly inside it without triggering immediate IDE syntax errors. To run imperative logic, you are naturally forced to wrap it in `script(() => { ... })`, keeping the execution graph safe and readable.

## 3. The never Type

**The Concern:** _How do I know which functions stop the JS execution?_

**The Reality:** Budment leverages TypeScript's `never` type to visualize its Panic-Recovery protocol.
Functions that yield thread control back to Go—such as `sleep()`, `barrier()`, `abort()`, and `fail()`—are typed as `never`. If you place a `sleep()` inside a JS hook, your IDE will automatically **gray out** any code below it, giving you real-time visual proof that the JS VM halts at that exact line.

## 4. Compile-Time Safeguards for AST Proxies

In Phase plan, data providers like `get()` or `env()` return AST Proxy objects, not real values. To prevent silent evaluation bugs during load tests, Budment traps the ECMAScript `Symbol.toPrimitive` lifecycle.

Applying arithmetic operators (`+`, `-`, `*`) or loose comparisons (`==`, `>`, `<`) to a Proxy outside a runtime hook triggers an immediate **Compile Error**.

TypeScript

```typescript
// COMPILE ERROR: Proxy cannot evaluate arithmetic outside a hook.
const nextPage = get("page") + 1;
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
