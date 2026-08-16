import { Node, HttpNode } from '../../pb/ast_schema';
import { Context, HttpRequest, HttpResponse } from '../../runtime';
import { HookRegistry } from './registry';
import { BuilderNode } from './types';

export class HttpBuilder implements BuilderNode {
    private ast: HttpNode;
    private uniqueId: string;

    constructor(method: string, url: string) {
        this.uniqueId = HookRegistry.generateNodeId('http');
        this.ast = {
            method: method,
            url: url,
            beforeHookId: '',
            afterHookId: ''
        };
    }

    before(callback: (ctx: Context, req: HttpRequest) => void): this {
        this.ast.beforeHookId = HookRegistry.register(callback, 'before');
        return this;
    }

    after(callback: (ctx: Context, req: HttpRequest, res: HttpResponse) => void): this {
        this.ast.afterHookId = HookRegistry.register(callback, 'after');
        return this;
    }

    build(): Node {
        return {
            id: this.uniqueId,
            http: this.ast
        };
    }
}

// Convenience helpers for HTTP builder initialization
export const http = {
    get: (url: string) => new HttpBuilder('GET', url),
    post: (url: string) => new HttpBuilder('POST', url),
    put: (url: string) => new HttpBuilder('PUT', url),
    delete: (url: string) => new HttpBuilder('DELETE', url)
};