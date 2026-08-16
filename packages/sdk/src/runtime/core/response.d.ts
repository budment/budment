/**
 * HTTP response context passed to `.after()` hooks.
 */
export interface HttpResponse {
    readonly status: number;
    readonly headers: Record<string, string>;
    readonly error?: string;

    readonly body: string;

    get<T = any>(path: string): T | undefined;
}