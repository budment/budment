import { http, sleep } from '@budment/sdk';

export const config = {
    // Dynamic load ramping profile
    stages: [
        { duration: "5s", target: 10 }, // Ramp-up to 10 VUs
        { duration: "10s", target: 10 }, // Stay at 10 VUs
        { duration: "5s", target: 0 },   // Ramp-down to 0 VUs
    ],
    thresholds: {
        "http_req_duration": "p95<1500ms",
    }
};

export default [
    http.get("https://httpbin.org/delay/0")
        .after({ expect: { status: 200 } }),
        
    sleep(0.2)
];