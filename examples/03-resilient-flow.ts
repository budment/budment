import { http, branch, poll, barrier, log, metrics, get } from '@budment/sdk';

export const config = {
    vus: 2,
    duration: "10s",
};

export const setup = [
    http.get("https://httpbin.org/get")
]

export default [
    // 1. Synchronize workers before starting the complex flow
    barrier("sync_start", { quorum: 2 }),
    log("All VUs have passed the synchronization barrier!"),

    http.get("https://httpbin.org/uuid")
        .after(
            { expect: { status: 200 }, extract: { "uuid": "request_id" } },
            // Uses standard template literals; resolves natively via Go tokens
            log(`Generated Request ID: ${get('request_id')}`) 
        ),

    // 2. Declarative branching based on extracted state
    branch(
        () => !!get("request_id"),
        [
            http.post("https://httpbin.org/anything")
                .before({
                    headers: { "Content-Type": "application/json" },
                    body: { tracking_id: get("request_id") }
                })
                .after({ expect: { status: 200 } })
        ],
        log("Bypassed downstream request: Missing tracking ID")
    ),

    // 3. Asynchronous polling (Do-While equivalent)
    poll(
        () => true, // Simulated condition: Loop until this returns true
        [
            http.get("https://httpbin.org/status/200")
                .after({ expect: { status: 200 } })
        ],
        { interval: "1000ms", maxAttempts: 2 }
    ),

    metrics.counter("completed_flows", 1)
];