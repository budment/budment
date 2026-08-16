/**
 * Global callback registry and deterministic ID generator.
 */

// Sequential counters to guarantee deterministic IDs across VM contexts
let hookCounter = 0;
let nodeCounter = 0;

export const HookRegistry = {
    hooks: new Map<string, Function>(),

    register(fn: Function, prefix: string = 'hook'): string {
        const id = `${prefix}_${hookCounter++}`;
        this.hooks.set(id, fn);
        return id;
    },

    generateNodeId(prefix: string): string {
        return `${prefix}_n${nodeCounter++}`;
    },

    getAll(): Record<string, Function> {
        return Object.fromEntries(this.hooks.entries());
    }
};

// Expose registry globally for Goja runtime context
(globalThis as any).HookRegistry = HookRegistry;