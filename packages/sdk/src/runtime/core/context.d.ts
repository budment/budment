/**
 * Shared memory access across workers backed and sceneries.
 */
export interface GlobalState {
    get<T = any>(key: string): T | undefined;    
    set<T = any>(key: string, value: T): void;    
    push<T = any>(queueName: string, value: T): void;
    pop<T = any>(queueName: string): T | undefined;
}

export interface SyncOptions {
    quorum?: number;
    gracePeriod?: string;
    maxWait?: string;
}

export interface RetryOptions {
    scope?: 'hook' | 'node';
    delay?: string;
}

/**
 * Execution context for worker iterations.
 */
export interface Context {
    readonly vuId: number;
    readonly iteration: number;
    readonly scenario: string;
    get<T = any>(key: string): T | undefined;
    set<T = any>(key: string, value: T): void;
    delete(key: string): void;

    log(msg: string, ...args: any[]): void;
    warn(msg: string, ...args: any[]): void;
    error(msg: string, ...args: any[]): void;
    
    tag(key: string, value: string): void;
    skip(reason?: string): void;
    retry(options?: RetryOptions): void;
    abort(reason?: string): void;
    fail(reason?: string): void;
    sleep(ms: number): void;    
    sync(name: string, options?: SyncOptions): void;

    readonly local: GlobalState;
    readonly global: GlobalState;
}

/**
 * Scenario setup hook context.
 */
export interface SetupContext {
    distribute(key: string, items: any[], fallback?: any): void;
    distributeRandom(key: string, items: any[]): void;
}