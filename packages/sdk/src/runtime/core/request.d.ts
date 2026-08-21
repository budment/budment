/**
 * Mutable request context passed to `.before()` hooks.
 */
export interface HttpRequest {
    readonly url: string;
    readonly method: string;
    
    /**
     * Extracts values from a JSON request using GJSON path syntax.
     * 
     * @param path GJSON path (e.g., "data.user.id" or "items.#.name")
     */
    get<T = any>(path: string): T | undefined;
    getBody(): object | string;

    /** Overrides the request payload. Objects are automatically marshaled to JSON. */
    setBody(data: object | string): void;

    /** Sets or overrides an HTTP header. */
    setHeader(key: string, val: string): void;

    /** Replaces a path template variable (e.g., {id}) in the URL. */
    setPath(key: string, value: string): void;

    /** Appends a URL-encoded query parameter. */
    setQuery(key: string, value: string): void;
}