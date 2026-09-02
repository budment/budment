import { Node, MatchNode } from '../pb/ast_schema';
import { HookRegistry } from '../builder/registry';
import { Pipeline } from '../builder/pipeline';
import { BuilderNode } from '../builder/types';

export type NodeInput = BuilderNode | BuilderNode[];

/**
 * Match node builder for multi-way branching (switch/case).
 */
export class MatchBuilder implements BuilderNode {
    private uniqueId: string;
    private conditionHookId: string;
    private casePipelines: Record<string, Pipeline> = {};
    private defaultPipeline?: Pipeline;

    constructor(
        condition: () => string | number,
        cases: Record<string | number, NodeInput>,
        defaultPath?: NodeInput,
    ) {
        this.uniqueId = HookRegistry.generateNodeId("match");
        this.conditionHookId = HookRegistry.register(this.uniqueId, "cond", condition);

        for (const [key, nodesOrNode] of Object.entries(cases)) {
            const pipeline = new Pipeline();
            const nodes = Array.isArray(nodesOrNode) ? nodesOrNode : [nodesOrNode];
            pipeline.add(...nodes);
            this.casePipelines[String(key)] = pipeline;
        }

        if (defaultPath) {
            this.defaultPipeline = new Pipeline();
            const nodes = Array.isArray(defaultPath) ? defaultPath : [defaultPath];
            if (nodes.length > 0) {
                this.defaultPipeline.add(...nodes);
            }
        }
    }

    build(): Node {
        const compiledCases: Record<string, any> = {};
        for (const [key, pipeline] of Object.entries(this.casePipelines)) {
            compiledCases[key] = pipeline.build();
        }

        return {
            id: this.uniqueId,
            match: {
                conditionHookId: this.conditionHookId,
                cases: compiledCases,
                defaultPath: this.defaultPipeline ? this.defaultPipeline.build() : undefined,
            } as MatchNode
        };
    }
}

export function match(
    condition: () => string | number,
    cases: Record<string | number, NodeInput>,
    defaultPath?: NodeInput,
) {
    return new MatchBuilder(condition, cases, defaultPath);
}