import { Node,PollNode, PollPolicy } from "../../pb/ast_schema";
import { HookRegistry } from "../core/registry";
import { Pipeline } from "../core/pipeline";
import { BuilderNode } from "../core/types";
import { Context } from '../../runtime/core/context';

/**
 * Poll builder for asynchronous polling loops.
 */
export class PollBuilder implements BuilderNode {
    private uniqueId: string;
    private conditionHookId: string;
    private logicPipeline = new Pipeline();
    private policy?: PollPolicy;

    constructor(
        condition: (ctx: Context) => boolean,
        logicPath: BuilderNode[],
        policy?: PollPolicy,
    ) {
        this.uniqueId = HookRegistry.generateNodeId("poll");
        this.conditionHookId = HookRegistry.register(this.uniqueId, "cond", condition);
        this.logicPipeline.add(...logicPath);
        this.policy = policy;
    }

    build(): Node {
        return {
            id: this.uniqueId,
            poll: {
                conditionHookId: this.conditionHookId,
                logic: this.logicPipeline.build(),
                policy: this.policy,
            } as PollNode
        };
    }
}

export function poll(
    condition: (ctx: Context) => boolean,
    logicPath: BuilderNode[],
    policy?: PollPolicy,
) {
    return new PollBuilder(condition, logicPath, policy);
}