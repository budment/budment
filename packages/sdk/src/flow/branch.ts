import { Node, BranchNode } from '../pb/ast_schema';
import { HookRegistry } from '../builder/registry';
import { Pipeline } from '../builder/pipeline';
import { BuilderNode } from '../builder/types';

export type NodeInput = BuilderNode | BuilderNode[];

/**
 * Branch builder for conditional execution (If/Else).
 */
export class BranchBuilder implements BuilderNode {
    private uniqueId: string;
    private conditionHookId: string;
    private truePipeline = new Pipeline();
    private falsePipeline?: Pipeline;

    constructor(
        condition: () => boolean,
        truePath: NodeInput,
        falsePath?: NodeInput
    ) {
        this.uniqueId = HookRegistry.generateNodeId('branch');
        this.conditionHookId = HookRegistry.register(this.uniqueId, "cond", condition);

        const trueNodes = Array.isArray(truePath) ? truePath : [truePath];
        this.truePipeline.add(...trueNodes);

        if (falsePath) {
            const falseNodes = Array.isArray(falsePath) ? falsePath : [falsePath];
            if (falseNodes.length > 0) {
                this.falsePipeline = new Pipeline();
                this.falsePipeline.add(...falseNodes);
            }
        }
    }

    build(): Node {
        return {
            id: this.uniqueId,
            branch: {
                conditionHookId: this.conditionHookId,
                truePath: this.truePipeline.build(),
                falsePath: this.falsePipeline ? this.falsePipeline.build() : undefined
            } as BranchNode
        };
    }
}

export function branch(
    condition: () => boolean,
    truePath: NodeInput,
    falsePath?: NodeInput
) {
    return new BranchBuilder(condition, truePath, falsePath);
}