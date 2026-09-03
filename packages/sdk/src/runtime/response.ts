/**
 * HTTP response context passed to `.after()` hooks.
 */
export interface HttpResponse {
    /** 
     * HTTP status code (e.g., 200, 404). 
     * Returns 0 for network or physical layer failures.
     */
    readonly status: number;

    /** Read-only map of merged response headers. */
    readonly headers: Record<string, string>;

    /** Infrastructure, timeout, or network error message, if any. */
    readonly error?: string;

    /** 
     * Extracts values from the JSON response using GJSON syntax,
     * or parses the entire response body when called without arguments.
     * 
     * @example
     * const id = res.json<number>("data.user.id");
     * const all = res.json();
     */
    json<T = any>(selector?: string): T | undefined;
}