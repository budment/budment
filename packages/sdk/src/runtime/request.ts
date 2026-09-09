/**
 * Dynamic options passed to request mutate method.
 */
export interface RequestOptions {
    /** Headers to inject or override. Values can be strings, numbers, or booleans. */
    headers?: Record<string, any>;

    /** Overrides the URL path (e.g. "/api/v2/login") or the entire target URL. */
    path?: string;

    /** URL query parameters to append or update. */
    params?: Record<string, any>;
}

/**
 * Internal descriptor marking payload data as a multipart file upload.
 */
export interface FileData {
    readonly __blaster_file: true;
    readonly data: ArrayBuffer | Uint8Array | string;
    readonly filename: string;
    readonly contentType: string;
}

/**
 * Mutable request context passed to `.before()` hooks.
 */
export interface HttpRequest {
    /** Target URL of the request */
    readonly url: string;

    /** HTTP Method (GET, POST, PUT, DELETE, ...) */
    readonly method: string;

    /**
     * Overrides request payload and configuration options (headers, path, query params).
     * Objects are automatically serialized to JSON.
     * 
     * @example
     * req.set({ user: "admin" }, { 
     *     headers: { Authorization: "Bearer token" },
     *     params: { retry: 1 } 
     * });
     */
    set(body: object | string | ArrayBuffer | Uint8Array, options?: RequestOptions): void;

    /**
     * Wraps binary or text data into a multipart file descriptor for form uploads.
     * 
     * @param data Raw file content as an ArrayBuffer, Uint8Array, or string.
     * @param filename Target file name (defaults to "upload.bin").
     * @param contentType MIME type of the file (defaults to "application/octet-stream").
     * 
     * @example
     * req.set({
     *     file: req.file(pdfBuffer, "invoice.pdf", "application/pdf")
     * });
     */
    file(
        data: ArrayBuffer | Uint8Array | string,
        filename?: string,
        contentType?: string
    ): FileData;
    /**
     * Extracts values from the request JSON payload using GJSON syntax,
     * or parses the whole body when called without arguments.
     * 
     * @param selector GJSON selector (e.g. "data.user.id"). If omitted, parses full payload.
     */
    json<T = any>(selector?: string): T | undefined;
}