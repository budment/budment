/**
 * Core execution environment interfaces for the Goja Virtual Machine.
 */

export interface SharedState {
    /** Retrieves a value by key. */
    get<T = any>(key: string): T | undefined;
    /** Stores a value by key. */
    set<T = any>(key: string, value: T): void;
    /** Pushes a value to a distributed FIFO queue. */
    push<T = any>(queueName: string, value: T): void;
    /** Pops a value from a distributed FIFO queue. Suspends if empty. */
    pop<T = any>(queueName: string): T | undefined;
}

export interface SyncOptions {
    /** Minimum number of workers required to release the synchronization barrier. */
    quorum?: number;
    /** Additional wait time (in ms) allowed for other workers after quorum is reached. */
    gracePeriod?: number;
    /** Maximum time to wait (in ms) from the first arrival before aborting the sync. */
    maxWait?: number;
}

export interface RetryOptions {
    /** Target scope for the retry operation. @default 'hook' */
    scope?: 'hook' | 'node';
    /** Delay before retrying in milliseconds. */
    delay?: number;
    /** 
     * Maximum number of retry attempts to prevent infinite loops. 
     * If exceeded, the engine will mark the step as failed and proceed to the next node.
     * @default 3
     */
    maxAttempts?: number;
}

/**
 * Execution Context for individual Virtual Users (Workers).
 * State mutations here are isolated and cleared after each iteration.
 */
export interface Context {
    // --- Metadata ---
    /** Unique numeric identifier for the current Virtual User. */
    readonly vuId: number;
    /** Current iteration count for this worker (0-indexed). */
    readonly iteration: number;
    /** Name of the currently executing scenario. */
    readonly scenario: string;

    // --- Worker-Scoped State ---
    /** 
     * Flat key-value store. 
     * Do NOT use dot-notation keys (e.g., "user.id"). Store the entire object instead.
     */
    get<T = any>(key: string): T | undefined;
    set<T = any>(key: string, value: T): void;
    delete(key: string): void;

    // --- Logging & Metrics ---
    log(msg: string, ...args: any[]): void;
    warn(msg: string, ...args: any[]): void;
    error(msg: string, ...args: any[]): void;

    /** Attaches a custom metric label to the current worker execution. */
    tag(key: string, value: string): void;

    // --- Flow Control ---
    /** Skips the current node and proceeds to the next. Terminates current JS execution. */
    skip(reason?: string): never;

    /** Suspends the worker and retries the operation. Terminates current JS execution. */
    retry(options?: RetryOptions): never;

    /** Aborts the current iteration and marks it as failed. Terminates current JS execution. */
    abort(reason?: string): never;

    /** Records a logical failure metric without halting execution. */
    fail(reason?: string): void;

    /** Returns the VM to the pool and suspends the worker. Terminates current JS execution. */
    sleep(ms: number): never;

    /** 
     * Pauses the worker at a synchronization barrier. 
     * Terminates current JS execution and returns VM to the pool.
     */
    sync(name: string, options?: SyncOptions): never;

    // --- Shared Memory ---
    /** Memory shared across all workers within the current Scenario. */
    readonly local: SharedState;
    /** Memory shared globally across all Scenarios in the Engine. */
    readonly global: SharedState;
}

/**
 * Scenario setup hook context.
 * Exclusively available during the scenario .setup() phase.
 */
export interface SetupContext {
    /** Distributes array elements to workers using Round-Robin. */
    distribute(key: string, items: any[], fallback?: any): void;

    /** Distributes array elements to workers randomly. */
    distributeRandom(key: string, items: any[]): void;

    /** Read/Write access to Scenario-scoped shared memory. */
    readonly local: SharedState;

    /** Read/Write access to Engine-scoped shared memory. */
    readonly global: SharedState;
}