/**
 * Global callback registry and deterministic ID generator.
 */

let nodeCounter = 0;

export const HookRegistry = {
    hooks: new Map<string, Function>(),

    generateNodeId(prefix: string): string {
        return `${prefix}_${++nodeCounter}`;
    },

    register(nodeId: string, hookType: string, fn: Function): string {
        const id = `${nodeId}_${hookType}`; 
        this.hooks.set(id, fn);
        return id;
    },

    getAll(): Record<string, Function> {
        return Object.fromEntries(this.hooks.entries());
    }
};

// Expose registry globally for Goja runtime context
(globalThis as any).HookRegistry = HookRegistry;