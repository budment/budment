import { MatchNode, Node, Pipeline as ProtoPipeline } from "../../pb/ast_schema";
import { HookRegistry } from "../core/registry";
import { Pipeline } from "../core/pipeline";
import { BuilderNode } from "../core/types";
import { Context } from '../../runtime/core/context';

/**
 * Match node builder for multi-way branching (switch/case).
 */
export class MatchBuilder implements BuilderNode {
    private uniqueId: string;
    private conditionHookId: string;
    private casePipelines: Record<string, Pipeline> = {};
    private defaultPipeline?: Pipeline;

    constructor(
        condition: (ctx: Context) => string | number,
        cases: Record<string | number, BuilderNode[]>,
        defaultPath?: BuilderNode[],
    ) {
        this.uniqueId = HookRegistry.generateNodeId("match");
        this.conditionHookId = HookRegistry.register(condition, "match_cond");

        for (const [key, nodes] of Object.entries(cases)) {
            const pipeline = new Pipeline();
            pipeline.add(...nodes);
            this.casePipelines[String(key)] = pipeline; 
        }

        if (defaultPath && defaultPath.length > 0) {
            this.defaultPipeline = new Pipeline();
            this.defaultPipeline.add(...defaultPath);
        }
    }

    build(): Node {
        const compiledCases: Record<string, ProtoPipeline> = {};
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
    condition: (ctx: Context) => string | number,
    cases: Record<string | number, BuilderNode[]>,
    defaultPath?: BuilderNode[],
) {
    return new MatchBuilder(condition, cases, defaultPath);
}