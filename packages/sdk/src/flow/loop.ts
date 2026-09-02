import { Node, LoopNode } from "../pb/ast_schema";
import { HookRegistry } from "../builder/registry";
import { Pipeline } from "../builder/pipeline";
import { BuilderNode } from "../builder/types";

export type LoopRangeConfig = { from: number; to: number };
export type ArrayHookCondition = () => any[]; 
export type NodeInput = BuilderNode | BuilderNode[];

/**
 * Loop builder supporting fixed counts, numeric ranges, or dynamic array getters.
 */
export class LoopBuilder implements BuilderNode {
    private uniqueId: string;
    private logicPipeline = new Pipeline();
    private loopConfig: Partial<LoopNode> = {};

    constructor(
        config: number | LoopRangeConfig | ArrayHookCondition,
        logicPath: NodeInput, 
    ) {
        this.uniqueId = HookRegistry.generateNodeId("loop");
        
        const nodes = Array.isArray(logicPath) ? logicPath : [logicPath];
        this.logicPipeline.add(...nodes);

        if (typeof config === "function") {
            this.loopConfig.arrayHookId = HookRegistry.register(this.uniqueId, "run", config);
        }
        else if (typeof config === "number") {
            this.loopConfig.count = config;
        }
        else if (typeof config === "object" && config !== null && "from" in config) {
            this.loopConfig.range = config;
        }
    }

    build(): Node {
        return {
            id: this.uniqueId,
            loop: {
                ...this.loopConfig,
                logic: this.logicPipeline.build(),
            } as LoopNode 
        };
    }
}

export function loop(count: number, logicPath: NodeInput): LoopBuilder;
export function loop(range: LoopRangeConfig, logicPath: NodeInput): LoopBuilder;
export function loop(arrayGetter: ArrayHookCondition, logicPath: NodeInput): LoopBuilder;

export function loop(
    config: number | LoopRangeConfig | ArrayHookCondition,
    logicPath: NodeInput,
): LoopBuilder {
    return new LoopBuilder(config, logicPath);
}