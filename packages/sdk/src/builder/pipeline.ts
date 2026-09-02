import { Node, Pipeline as ProtoPipeline } from "../pb/ast_schema";
import { BuilderNode } from "./types";

/**
 * Container for executing sequential AST steps.
 */
export class Pipeline {
    private nodes: Node[] = [];

    /**
     * Build and append step nodes to the pipeline sequence.
     */
    add(...builders: BuilderNode[]): this {
        for (const builder of builders) {
            this.nodes.push(builder.build());
        }
        return this;
    }

    build(): ProtoPipeline {
        return {
            steps: this.nodes,
        };
    }
}
