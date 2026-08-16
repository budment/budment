export interface RandomUtils {
    /** Generates a fast UUID v4 via the native Go engine. */
    uuid(): string;
    
    /** Generates a random integer between min and max (inclusive). */
    integer(min: number, max: number): number;
    
    /** Selects a random element from the provided array. */
    pick<T>(array: T[]): T;
    
    /** Generates a random alphanumeric string of the specified length. */
    string(length: number): string;
}

/**
 * Random data generation utilities backed by Go native methods.
 */
export declare const random: RandomUtils;