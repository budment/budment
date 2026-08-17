import { Node, ScriptNode } from "../../pb/ast_schema";
import { HookRegistry } from "./registry";
import { BuilderNode } from '../core/types';

/**
 * Script execution node builder.
 */
export class ScriptBuilder implements BuilderNode {
    private hookId: string;
    private uniqueId: string;

    constructor(callback: (ctx: any) => void) {
        this.uniqueId = HookRegistry.generateNodeId("script");
        this.hookId = HookRegistry.register(callback, "script");
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

export function script(callback: (ctx: any) => void) {
    return new ScriptBuilder(callback);
}