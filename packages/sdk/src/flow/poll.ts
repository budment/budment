import { Node, PollNode, PollPolicy } from "../pb/ast_schema";
import { HookRegistry } from "../builder/registry";
import { Pipeline } from "../builder/pipeline";
import { BuilderNode } from "../builder/types";

export type NodeInput = BuilderNode | BuilderNode[];

/**
 * Poll builder for asynchronous polling loops.
 */
export class PollBuilder implements BuilderNode {
    private uniqueId: string;
    private conditionHookId: string;
    private logicPipeline = new Pipeline();
    private policy?: PollPolicy;

    constructor(
        condition: () => boolean,
        logicPath: NodeInput,
        policy?: PollPolicy,
    ) {
        this.uniqueId = HookRegistry.generateNodeId("poll");
        this.conditionHookId = HookRegistry.register(this.uniqueId, "cond", condition);

        const nodes = Array.isArray(logicPath) ? logicPath : [logicPath];
        this.logicPipeline.add(...nodes);

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

export function poll(condition: () => boolean, logicPath: NodeInput, policy?: PollPolicy) {
    return new PollBuilder(condition, logicPath, policy);
}