export interface Expectation<T = any> {
    toBe(expected: T): void;
    toBeGreaterThan(expected: number): void;
    toBeLessThan(expected: number): void;
    toContain(expected: any): void;
    toBeDefined(): void;
    
    readonly not: Expectation<T>;
}

/**
 * Soft assertion utility for validating values without halting worker execution.
 * Failures are logged and tracked in execution metrics.
 */
export declare function expect<T = any>(value: T): Expectation<T>;