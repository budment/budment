/**
 * HTTP response context passed to `.after()` hooks.
 */
export interface HttpResponse {
    /** 
     * HTTP status code (e.g., 200, 404). 
     * Returns 0 for network or physical layer failures.
     */
    readonly status: number;
    
    /** Read-only map of response headers. */
    readonly headers: Record<string, string>;
    
    /** Infrastructure or network error message, if any. */
    readonly error?: string;

    /**
     * Returns the full response body as a string. 
     * This allocates memory in the JS VM. Use only for HTML, XML, or Text parsing.
     */
    readonly body: string;

    /**
     * Extracts values from a JSON response using GJSON path syntax.
     * Operates directly on native byte slices (Zero-Allocation).
     * 
     * @param path GJSON path (e.g., "data.user.id" or "items.#.name")
     */
    get<T = any>(path: string): T | undefined;
}