import { Node, LoopNode } from "../../pb/ast_schema";
import { HookRegistry } from "../core/registry";
import { Pipeline } from "../core/pipeline";
import { BuilderNode } from "../core/types";
import { Context } from '../../runtime/core/context';

export type LoopRangeConfig = { from: number; to: number };
export type ArrayHookCondition = (ctx: Context) => any[];

/**
 * Root builder for scenario definition and configuration.
 */
export class LoopBuilder implements BuilderNode {
    private uniqueId: string;
    private logicPipeline = new Pipeline();
    private loopConfig: Partial<LoopNode> = {};

    constructor(
        config: number | LoopRangeConfig | ArrayHookCondition,
        logicPath: BuilderNode[],
    ) {
        this.uniqueId = HookRegistry.generateNodeId("loop");
        this.logicPipeline.add(...logicPath);

        if (typeof config === "function") {
            this.loopConfig.arrayHookId = HookRegistry.register(config, "loop_array");
        }
        else if (typeof config === "number") {
            this.loopConfig.count = config;
        }
        else if (
            typeof config === "object" &&
            config !== null &&
            "from" in config
        ) {
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

export function loop(count: number, logicPath: BuilderNode[]): LoopBuilder;
export function loop(
    range: LoopRangeConfig,
    logicPath: BuilderNode[],
): LoopBuilder;
export function loop(
    arrayGetter: ArrayHookCondition,
    logicPath: BuilderNode[],
): LoopBuilder;

export function loop(
    config: number | LoopRangeConfig | ArrayHookCondition,
    logicPath: BuilderNode[],
): LoopBuilder {
    return new LoopBuilder(config, logicPath);
}
