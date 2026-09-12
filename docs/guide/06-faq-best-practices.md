---
title: FAQs, Anti-Patterns & Best Practices
description: Understand the Budment philosophy, hardware realities of the JS VM pool, syntax safeguards, and the golden rule of test authoring.
---

# FAQs, Anti-Patterns & Best Practices

Understanding how Budment balances declarative topology with imperative JavaScript hooks is key to mastering the engine. This guide addresses common developer concerns, explains the hardware realities of the Virtual Machine (VM) pool, and outlines the core rules for writing efficient scenarios.

## 1. The Golden Rule: DSL vs. JS Hooks

If you only remember one rule when writing Budment scenarios, it should be this:

> **Outside `() => {}` is the Static DSL. Inside `() => {}` is the Runtime Hook.**

* **The DSL (Static Array):** Used to define the *structure* of your test (e.g., HTTP requests, branches, loops).
* **The JS Hook:** Used for complex logic. Any math (`Date.now()`), complex object destructuring, conditional data mutations, or cryptographic hashing **must** be placed inside a JavaScript hook (`script(() => { ... })` or `.before(req => { ... })`).

## 2. The "God Hook" Myth & Concurrency Optimization

**The Concern:** *If I put too much logic into JS hooks, won't it create a bottleneck, block the engine, and defeat the purpose of Go's high concurrency?*

**The Reality:** Writing robust logic inside JS hooks is not a "dirty" anti-pattern—it is exactly how the engine is designed to be used safely and efficiently. Borrowing a VM from the pool is highly optimized for performance:

* **Hardware Reality Check:** Even if you spawn 10,000 Virtual Users (goroutines), if your machine only has a 4-core CPU, it can only physically execute 4 instructions in parallel at any exact nanosecond. The Go scheduler efficiently multiplexes the rest.
* **Reducing Context Switching & GC Pressure:** JavaScript is inherently single-threaded. If Budment spawned 10,000 isolated JS VMs, the CPU would drown in OS context switches and Garbage Collection (GC) pauses. By using a warm `sync.Pool`, Budment limits the active VMs to what the CPU can actually handle.
* **Speed via Warm Recycling:** When a worker finishes a hook, it immediately returns the VM to the pool. Subsequent workers reuse these "warm" VMs, which avoids memory allocation overhead and executes logic significantly faster than spinning up new context scopes.

## 3. Syntax Safeguards: You Can't "Break" the DSL

**The Concern:** *What if a developer accidentally writes standard JavaScript logic (like `let a = 1`, `if/else`, or `while` loops) directly in the DSL instead of a hook?*

**The Reality:** The Budment DSL is syntactically protected by standard JavaScript/TypeScript grammar. 

Your scenario is an array of object instances `[ node1, node2, node3 ]` separated by commas. You physically cannot write imperative statements like variable assignments (`const x = 10`) or `while` loops directly inside an array declaration without triggering immediate IDE syntax errors. 

If you need to execute imperative logic, you are naturally forced to wrap it in a function signature: `script(() => { const x = 10; })`. This elegantly protects the execution graph from invalid runtime logic while keeping the topology easy to read.

## 4. The Magic of the `never` Type

Budment leverages advanced TypeScript definitions to provide real-time visual feedback on how the engine handles thread yielding via its Panic-Recovery protocol.

Operational nodes that cause the JavaScript VM to immediately suspend and yield thread control back to Go—such as `sleep()`, `barrier()`, `abort()`, and `fail()`—are typed in the SDK to return `never`.

**Dual Behavior (DSL vs. Hook):**

* **In the DSL Array:** Because arrays are just lists of expressions, putting `sleep(1)` inside the DSL array works perfectly as a structural node. It will not gray out the next HTTP request in the list.
* **Inside a JS Hook:** If you place a `sleep()` or `abort()` inside a JS hook, the TypeScript compiler's Control Flow Analysis takes over. Your IDE will automatically **gray out** any code written below it.

```typescript
import { abort, get, script } from '@budment/sdk';

export default [
    script(() => {
        const balance = get<number>("balance");
        if (balance < 0) {
            // Triggers a native Go panic ("BUDMENT_ABORT") to yield the VM instantly
            abort("Negative balance detected.");
            
            // The IDE grays out this line automatically!
            // It visually tells the developer: "The JS VM is yielded here. This will never run."
            console.log("This is unreachable code."); 
        }
    })
];
```
This acts as a brilliant built-in safety mechanism, providing developers with immediate visual confirmation that the engine will halt the script and release the VM back to the pool without executing further.