import { Node, BranchNode } from '../pb/ast_schema';
import { HookRegistry } from '../builder/registry';
import { Pipeline } from '../builder/pipeline';
import { BuilderNode } from '../builder/types';

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
        truePath: BuilderNode[],
        falsePath?: BuilderNode[]
    ) {
        this.uniqueId = HookRegistry.generateNodeId('branch');
        this.conditionHookId = HookRegistry.register(this.uniqueId, "cond", condition);

        this.truePipeline.add(...truePath);

        if (falsePath && falsePath.length > 0) {
            this.falsePipeline = new Pipeline();
            this.falsePipeline.add(...falsePath);
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
    truePath: BuilderNode[],
    falsePath?: BuilderNode[]
) {
    return new BranchBuilder(condition, truePath, falsePath);
}