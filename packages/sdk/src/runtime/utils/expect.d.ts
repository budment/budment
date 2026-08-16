export interface Expectation<T = any> {
    toBe(expected: T): void;
    toBeGreaterThan(expected: number): void;
    toBeLessThan(expected: number): void;
    toContain(expected: any): void;
    toBeDefined(): void;
    
    /** Negates the assertion. */
    readonly not: Expectation<T>;
}

/**
 * Soft-assertion library for validating responses.
 * Failures are logged and recorded as metrics without crashing the worker.
 * 
 * @example
 * expect(res.status).toBe(200);
 * expect(res.get("role")).toBe("admin");
 */
export declare function expect<T = any>(value: T): Expectation<T>;