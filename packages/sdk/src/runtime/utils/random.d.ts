/**
 * Random data generation utilities backed by Go native methods.
 */
export interface RandomUtils {
    uuid(): string;  
    integer(min: number, max: number): number; 
    pick<T>(array: T[]): T;   
    string(length: number): string;
}

/**
 * Random data generation utilities backed by Go native methods.
 */
export declare const random: RandomUtils;