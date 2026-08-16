/**
 * Mutable request context passed to `.before()` hooks.
 */
export interface HttpRequest {
    readonly url: string;
    readonly method: string;
    
    setBody(data: object | string): void;
    setHeader(key: string, val: string): void;
    setPath(key: string, value: string): void;
    setQuery(key: string, value: string): void;
}   