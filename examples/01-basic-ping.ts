import { http, sleep, log, metrics, barrier } from '@budment/sdk';

export const config = {
    vus: 2,
    duration: "6s",
    thresholds: {
        "http_req_duration": "p95<2s",
        "http_req_failed": "rate<0.01",
    },
};

export default [
    http.get("https://httpbin.org/get")
        .after({ expect: { status: 200 } }),

    sleep(1.5),

    http.post("https://httpbin.org/post")
        .before(
            // Auto-serializes JavaScript objects to JSON
            {
                headers: { "Content-Type": "application/json" },
                body: { message: "Hello from Budment!" }
            },
            // Operational nodes can be chained sequentially in hooks
            barrier("sync_start", { quorum: 2 })
        )
        .after(
            { expect: { status: 200 } },
            // Imperative JS hook for custom logging
            res => log(`Response received with status: ${res.status}`)
        ),

    metrics.counter("successful_iterations", 1)
];