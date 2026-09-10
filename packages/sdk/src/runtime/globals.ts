/**
 * Core execution environment interfaces for the Goja Virtual Machine.
 */

export interface SharedState {
    /** Retrieves a value by key. */
    get<T = any>(key: string): T | undefined;
    /** Stores a value by key. */
    set<T = any>(key: string, val: T): void;
    /** Pushes a value to a distributed FIFO queue. */
    push(queue: string, val: any): void;
    /** Pops a value from a distributed FIFO queue. Suspends if empty. */
    pop<T = any>(queue: string): T | undefined;
}

export interface BarrierOptions {
    /** Minimum number of workers required to release the synchronization barrier. */
    quorum?: number;
    /** Additional wait time (in ms) allowed for other workers after quorum is reached. */
    gracePeriod?: number;
    /** Maximum time to wait (in ms) from the first arrival before aborting the barrier. */
    maxWait?: number;
}
export interface ExecutionInfo {
    /** Unique numeric identifier for the current Virtual User. */
    vuId: number | string;
    /** Current iteration count for this worker (0-indexed). */
    iteration: number;
    /** Name of the currently executing scenario. */
    scenario: string;
}

// Access globalThis with direct Go bridge bindings.
export const _g = globalThis as any;

// Safe fallback node & function to prevent "TypeError: ... is not a function"
export const dummyNode = { build: () => ({}) };
export const dummyFn: any = () => dummyNode;

/** Retrieves a value by key. */
export const get: <T = any>(key: string) => T | undefined = _g.get || dummyFn;

/** Stores a value by key. */
export const set: (key: string, val: any) => void = _g.set || dummyFn;

/** Reads a file from disk as a string (supports both 'open' and 'load'). */
export const open: (filepath: string, mode?: 'r' | 'b' | string) => any = _g.open || _g.load || dummyFn

/** Retrieves an environment variable set in Go runtime. */
export const env: (key: string, fallback?: string) => string = _g.env || ((_, f) => f || '');

/** Distributes array elements to workers using Round-Robin. */
export const distribute: (key: string, items: any[], fallback?: any) => void = _g.distribute || dummyFn;

export const log: (msg: string) => any = _g.log || dummyFn;
export const warn: (msg: string) => any = _g.warn || dummyFn;
export const error: (msg: string) => any = _g.error || dummyFn;

/** Attaches a custom metric label to the current worker execution. */
export const tag: (key: string, value: string) => any = _g.tag || dummyFn;

/** Aborts the current iteration and marks it as failed. Terminates current JS execution. */
export const abort: (reason?: string) => never = _g.abort || dummyFn;

/** Records a logical failure metric without halting execution. */
export const fail: (reason?: string) => any = _g.fail || dummyFn;

/** Returns the VM to the pool and suspends the worker. Terminates current JS execution. */
export const sleep: (duration_s: number) => never = _g.sleep || dummyFn;

/** 
 * Pauses the worker at a synchronization barrier. 
 * Terminates current JS execution and returns VM to the pool.
 */
export const barrier: (name: string, opts?: BarrierOptions) => never = _g.barrier || dummyFn;

/** Memory shared across all workers within the current Scenario. */
export const local: SharedState = _g.local || dummyNode;
/** Memory shared globally across all Scenarios in the Engine. */
export const global: SharedState = _g.global || dummyNode;

const _m = _g.metrics || {};
export const metrics = {
    /** Counts cumulative values, e.g. orders created. */
    counter: (_m.counter || dummyFn) as (name: string, value: number) => any,
    /** Tracks distributions such as latency and percentiles. */
    trend: (_m.trend || dummyFn) as (name: string, value: number) => any,
    /** Tracks the current value, e.g. CPU usage. */
    gauge: (_m.gauge || dummyFn) as (name: string, value: number) => any,
};

export const info: ExecutionInfo = _g.info || dummyNode;