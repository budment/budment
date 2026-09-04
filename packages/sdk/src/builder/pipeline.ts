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
    add(...builders: (BuilderNode | any)[]): this {
        for (const builder of builders) {
            if (builder && typeof builder.build === 'function') {
                const node = builder.build();
                if (node) {
                    this.nodes.push(node);
                }
            }
        }
        return this;
    }

    build(): ProtoPipeline {
        return {
            steps: this.nodes,
        };
    }
}