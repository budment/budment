import { Node, HttpNode, ReqMutateNode, ResAssertNode } from '../pb/ast_schema';
import { HttpRequest } from '../runtime/request';
import { HttpResponse } from '../runtime/response';
import { HookRegistry } from './registry';
import { BuilderNode } from './types';
import { Pipeline } from './pipeline';

class JSHookNode implements BuilderNode {
    constructor(private hookId: string) { }
    build(): Node {
        return { id: this.hookId, script: { hookId: this.hookId } };
    }
}

export interface StaticRequest {
    headers?: Record<string, string>;
    body?: string;
}

export interface StaticExpect {
    status?: number;
    bodyContains?: string;
}

export interface StaticAssert {
    expect?: StaticExpect;
    extract?: Record<string, string>;
}

export type NodeInput = BuilderNode | BuilderNode[];
export type BeforeAction = (((req: HttpRequest) => void) | NodeInput | StaticRequest);
export type AfterAction = (((res: HttpResponse, req: HttpRequest) => void) | NodeInput | StaticAssert);

export class HttpBuilder implements BuilderNode {
    private method: string;
    private url: string;
    private uniqueId: string;

    private beforePipeline = new Pipeline();
    private afterPipeline = new Pipeline();

    constructor(method: string, url: string) {
        this.uniqueId = HookRegistry.generateNodeId('http');
        this.method = method;
        this.url = url;
    }

    before(...args: BeforeAction[]): this {
        for (const arg of args) {
            if (typeof arg === 'function') {
                const hookId = HookRegistry.register(this.uniqueId, 'before', arg);
                this.beforePipeline.add(new JSHookNode(hookId));
            }
            else if (arg && typeof arg === 'object' && !Array.isArray(arg) && !('build' in arg)) {
                const reqNode = arg as StaticRequest;
                this.beforePipeline.add({
                    build: () => ({
                        id: HookRegistry.generateNodeId('req_mutate'),
                        reqMutate: {
                            headers: reqNode.headers || {},
                            body: reqNode.body || ""
                        } as ReqMutateNode
                    })
                });
            }
            else {
                const nodes = Array.isArray(arg) ? arg : [arg];
                this.beforePipeline.add(...nodes);
            }
        }
        return this;
    }

    after(...args: AfterAction[]): this {
        for (const arg of args) {
            if (typeof arg === 'function') {
                const wrapperFn = (req: HttpRequest, res: HttpResponse) => arg(res, req);
                const hookId = HookRegistry.register(this.uniqueId, 'after', wrapperFn);
                this.afterPipeline.add(new JSHookNode(hookId));
            }
            else if (arg && typeof arg === 'object' && !Array.isArray(arg) && !('build' in arg)) {
                const assertNode = arg as StaticAssert;
                this.afterPipeline.add({
                    build: () => ({
                        id: HookRegistry.generateNodeId('res_assert'),
                        resAssert: {
                            expectStatus: assertNode.expect?.status || 0,
                            expectBodyContains: assertNode.expect?.bodyContains || "",
                            extract: assertNode.extract || {}
                        } as ResAssertNode
                    })
                });
            }
            else {
                const nodes = Array.isArray(arg) ? arg : [arg];
                this.afterPipeline.add(...nodes);
            }
        }
        return this;
    }

    build(): Node {
        return {
            id: this.uniqueId,
            http: {
                method: this.method,
                url: this.url,
                before: this.beforePipeline.build(),
                after: this.afterPipeline.build()
            } as unknown as HttpNode
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