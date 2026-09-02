import { Node } from '../pb/ast_schema';

/**
 * Base interface for AST builders compiling to Protobuf nodes.
 */
export interface BuilderNode {
    build(): Node;
}