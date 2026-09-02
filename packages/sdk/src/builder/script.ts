import { Node, ScriptNode } from "../pb/ast_schema";
import { HookRegistry } from "./registry";
import { BuilderNode } from './types';

/**
 * Script execution node builder.
 */
export class ScriptBuilder implements BuilderNode {
    private hookId: string;
    private uniqueId: string;

    constructor(callback: () => void) {
        this.uniqueId = HookRegistry.generateNodeId("script");
        this.hookId = HookRegistry.register(this.uniqueId, "run", callback);
    }

    build(): Node {
        return {
            id: this.uniqueId,
            script: {
                hookId: this.hookId,
            } as ScriptNode
        };
    }
}

export function script(callback: () => void) {
    return new ScriptBuilder(callback);
}