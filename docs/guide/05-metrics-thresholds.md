---

title: Metrics, Custom Telemetry & SLA Thresholds
description: Complete guide to custom business metrics, worker tagging, failure classifications, and automated SLA Quality Gates.
---

# Metrics, Custom Telemetry & SLA Thresholds

Budment features a built-in observability subsystem engineered to tracks default HTTP transport and flows node metrics while allowing scripts to record custom domain metrics, tag virtual user executions, and enforce automated Quality Gates (SLAs) for CI/CD pipelines.

## 1. Custom Business Metrics

Beyond standard HTTP latency and request counts, scripts can record custom domain KPIs using the `metrics` primitive. These metrics appear directly in the terminal summary and exported reports.

```typescript
import { metrics, get, http } from 'budment';

export default [
    http.post("https://api.example.com/checkout")
        .after({
            expect: { status: 200 },
            extract: { "order.total": "cart_total" }
        }),

    // 1. Counter: Tracks cumulative sums
    metrics.counter("completed_orders", 1),

    // 2. Trend: Calculates distributions, averages, min, and max
    metrics.trend("cart_value_usd", get<number>("cart_total") || 0),

    // 3. Gauge: Tracks instantaneous current values
    metrics.gauge("last_processed_user", get<number>("user_id") || 0)
];
```

### Metric Types Reference

| **Metric Type** | **Method**                   | **Description**                                                                       | **Primary Use Cases**                                              |
| --------------- | ---------------------------- | ------------------------------------------------------------------------------------- | ------------------------------------------------------------------ |
| **Counter**     | `metrics.counter(name, val)` | Monotonically accumulating sum.                                                       | Total orders, completed iterations, business-level errors.         |
| **Trend**       | `metrics.trend(name, val)`   | Collects values and calculates percentile distributions (for reporting) and averages. | Processing durations, item checkout values, response body lengths. |
| **Gauge**       | `metrics.gauge(name, val)`   | Stores the latest value or instantaneous state.                                       | Memory consumption, queue depth, active user IDs.                  |

## 2. Worker Tagging: 

The `tag(key, value)` primitive attaches metadata labels to the active Virtual User goroutine. Tags are utilized for diagnostic tracing and filtering telemetry in structured debug logs.

**TypeScript**

```typescript
import { tag, http, get } from 'budment';

export default [
    http.post("https://api.example.com/auth/login")
        .after({ extract: { "tier": "account_tier" } }),

    // Attach account tier label to the active worker
    tag("user_tier", get<string>("account_tier") || "standard")
];
```

## 3. Standalone Script Nodes: 

When arbitrary JavaScript logic must execute sequentially between requests—without being attached to a specific HTTP `.before()` or `.after()` hook—wrap the logic using the `script(fn)` builder:

**TypeScript**

```typescript
import { script, set, log, metrics } from 'budment';

export default [
    script(() => {
        const nonce = Date.now().toString(36);
        set("request_nonce", nonce);
        log(`Generated session nonce: ${nonce}`);
        metrics.counter("nonces_generated", 1);
    })
];
```

## 4. Failure Classifications:

Budment differentiates between non-fatal logic violations and critical errors that require terminating an iteration immediately:

**TypeScript**

```typescript
import { fail, abort, get, script } from 'budment';

export default [
    script(() => {
        const balance = get<number>("account_balance") || 0;

        // 1. Non-fatal failure: Increments error count, pipeline continues executing
        if (balance < 100) {
            fail("Low balance warning: Account balance is below recommended reserve.");
        }

        // 2. Fatal iteration abort: Halts current iteration immediately
        if (balance < 0) {
            abort("Critical violation: Negative account balance detected. Discarding remaining steps.");
        }
    })
];
```

### Comparison Matrix

| **Mechanism**   | **Behavior on Worker**                                              | **Metrics Impact**           | **Next Step**                           |
| --------------- | ------------------------------------------------------------------- | ---------------------------- | --------------------------------------- |
| `fail(reason)`  | Continues execution to the next node in the pipeline.               | Increments `LogicFailCount`. | Next step in current pipeline.          |
| `abort(reason)` | Immediately stops the current iteration via runtime panic-recovery. | Increments `LogicFailCount`. | Starts next iteration (VU scope reset). |

## 5. Automated SLA Quality Gates 

Quality Gates define pass/fail criteria for your system under test. When one or more thresholds are breached, Budment exits with a non-zero code, automatically failing CI/CD pipeline jobs.

### Declaring Thresholds

**TypeScript**

```typescript
export const config = {
    vus: 20,
    duration: "1m",
    thresholds: {
        // Standard HTTP Latency thresholds
        "http_req_duration": "p95<300ms", // 95th percentile under 300ms
        "p99": "<800ms",                  // Direct alias usage

        // Failure rate thresholds (supports both decimals and percentages)
        "http_req_failed": "rate<0.01",   // Failure rate must be under 1%
        "fail_rate": "<= 2%",             // Alternative percentage notation

        // Custom Metrics thresholds (Do NOT use prefixes like "count>" or "p95>")
        "completed_orders": ">100",       // Evaluates the Counter's Total Sum
        "cart_value_usd": ">50"           // Evaluates the Trend's Average
    }
};
```

### Custom Metrics Value Resolution

When defining thresholds for custom metrics, **you must use the operator directly** (e.g., `>100`, `<=50`). Do not prefix the condition with words like `count>` or `p95>`, as the engine's parser will misinterpret the metric name.

The engine resolves the actual evaluation value based on the custom metric type:

* **Counter:** Evaluates the **Sum** (total accumulated value).
* **Gauge:** Evaluates the **Last** (most recently recorded value).
* **Trend:** Evaluates the **Average** (`Sum / Count`). *(Note: Percentile thresholding like* *`p95`* *is currently not supported for Custom Trends in SLAs)*.

### Complete Threshold Criteria Reference

| **Target Metric**     | **Supported Syntax & Aliases**               | **Meaning**                                                | **Example Criteria**                      |
| --------------------- | -------------------------------------------- | ---------------------------------------------------------- | ----------------------------------------- |
| **p95 Latency**       | `http_req_duration`, `p95`, `latency_p95`    | 95th percentile response duration.                         | `p95<300ms` or `<300ms` (if key is `p95`) |
| **p99 Latency**       | `p99`, `latency_p99`                         | 99th percentile response duration.                         | `p99<500ms`                               |
| **p50 / p90**         | `p50`, `p90`, `latency_p90`                  | Median and 90th percentile durations.                      | `p90<200ms`                               |
| **Max / Min Latency** | `max`, `latency_max`, `min`                  | Absolute boundaries across all requests.                   | `max<5s`, `min<50ms`                      |
| **Failure Rate**      | `http_req_failed`, `fail_rate`, `error_rate` | Ratio of failed requests (network + logic).                | `rate<0.01`, `<= 1.5%`                    |
| **Iterations**        | `iterations`                                 | Total completed worker iteration passes.                   | `>= 500`                                  |
| **Total Requests**    | `requests`                                   | Total volume of network executions completed.              | `>= 1000`                                 |
| **Custom Metrics**    | `<custom_metric_name>`                       | Evaluates Average (Trend), Sum (Counter), or Last (Gauge). | `>500`, `<=100`                           |

### Supported Operators & Units

* **Operators:** `<`, `<=`, `>`, `>=`, `==`, `!=`
* **Duration Units:** `ms` (milliseconds), `s` (seconds), `m` (minutes). E.g., `500ms`, `2s`, `1m`.
* **Percentage Units:** `%` (e.g., `1%`, `5.5%`) or fractional decimals (`0.01` for 1%).
