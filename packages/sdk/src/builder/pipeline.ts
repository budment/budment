import { Node, Pipeline as ProtoPipeline } from '../pb/ast_schema';
import { BuilderNode } from './types';

/**
 * Container for executing sequential AST steps.
 * Automatically resolves and flattens nested arrays of builder nodes.
 */
export class Pipeline {
    private nodes: Node[] = [];

    /**
     * Build and append step nodes to the pipeline sequence.
     * Recursively unpacks nested arrays to prevent silent step drops.
     */
    add(...builders: any[]): this {
        const unpack = (items: any[]) => {
            for (const item of items) {
                if (item === null || item === undefined) {
                    continue;
                }

                if (Array.isArray(item)) {
                    unpack(item);
                } else if (typeof item.build === 'function') {
                    const node = item.build();
                    if (node && typeof node === 'object' && node.id) {
                        this.nodes.push(node);
                    }
                } else if (typeof item === 'object' && item.id) {
                    // Direct protobuf node injection
                    this.nodes.push(item as Node);
                }
            }
        };

        unpack(builders);
        return this;
    }

    build(): ProtoPipeline {
        return {
            steps: this.nodes
        };
    }
}