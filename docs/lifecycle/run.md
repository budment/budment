# EXECUTION LIFECYCLE SPECIFICATION

**Main Command:** `blaster run` 

This document outlines the complete lifecycle of a load-testing session (from initialization to report generation), corresponding to the core architecture located in the `internal/runtime/` directory.

---

## PHASE 1: INITIALIZATION & CONFIGURATION LOADING (INIT & LOAD CONFIG)
*(Handled by: `internal/config/`)*

This phase acts as the "gatekeeper", responsible for loading and verifying the integrity of all inputs. If any validation fails, the program triggers a Fail-Fast mechanism and aborts immediately.

*   **Read Configuration:** Blaster locates and parses the `blaster.yaml` file in the current working directory (or the path specified via the `--config` flag).
*   **Hierarchy of Precedence:** Default values $\rightarrow$ YAML Config $\rightarrow$ Environment Variables (`.env`) $\rightarrow$ CLI Flags. *(CLI flags have the highest priority).*
*   **Validation:** Ensures all configurations are valid (e.g., URL reachability, cache directory read/write permissions, non-empty tokens, properly formatted endpoint paths).
*   **Initialize Global Context:** Instantiates a `RuntimeContext` object in memory (`internal/runtime/context.go`). This context holds the parsed configurations, login states, token lists, and translates Endpoints into executable `Task` structures (containing URL, Method, Headers, Payload Templates) for safe cross-thread data sharing.

---

## PHASE 2: PLANNING & ORCHESTRATION (PLANNER & DAG)
*(Handled by: `internal/planner/` and `internal/cache/`)*

This is the "brain" of the engine. No HTTP requests are fired yet; instead, it resolves complex data dependencies to formulate a comprehensive execution plan.

*   **Scope Determination:** Evaluates `blaster/cached` to identify APIs affected by recent code changes (Affected APIs) for prioritization. Checks if Long-lived Tokens were provided by the user to either skip or trigger Phase 3 (Authentication).
*   **Dependency Graph Construction (DAG):** The Planner builds a Directed Acyclic Graph based on multiple sources: Endpoint configurations, naming conventions, pre-seeded data, and user-declared dependencies. *Note: If insufficient information is provided, the Endpoint is treated as independent.*
*   **Topological Sort:**
    *   **Level 1:** Independent APIs (e.g., Create User, Create Product, Create Category).
    *   **Level 2:** Dependent APIs requiring Level 1 data (e.g., Create Order - requires `user_id` and `product_id`).
    *   **Level 3:** Subsequent or high-level actions (e.g., Payment Checkout, Cancel Order).
*   **Task Generation:** The execution plan is locked. All subsequent phases treat this plan as the immutable source of truth and execute it mechanically.

---

## PHASE 3: AUTHENTICATION (AUTH HOOK)
*(Handled by: `internal/auth/`)*

*(This phase is bypassed if the user explicitly injects Long-lived Tokens via environment variables or the YAML configuration).*

*   **Warm-up Worker:** If `auth.accounts` and `auth.login` are configured, Blaster spawns a single request to hit the login endpoint.
*   **Extraction & Injection:** Utilizes `JSONPath` to extract the Access Token from the HTTP Response, formats it (e.g., `Bearer <token>`), and persists it in the `RuntimeContext` to be injected into all subsequent requests.

---

## PHASE 3.5: RECURSIVE DATA SEEDING
*(Resolves Data Contention & Metric Pollution)*

Instead of allowing hundreds of workers to recursively call APIs (which causes network bottlenecks and skews latency metrics), Blaster dedicates a "Seeding" phase using **a single isolated thread**.

*   **Scan and Seed:** The system traverses Level 1 APIs (e.g., `GET /category`, `GET /product`). If the existing data volume is insufficient, it recursively triggers `POST` requests multiple times to "seed" the database.
*   **Thread-Safe Storage:** Extracts all generated Primary Keys (`id`) and pushes them into Go Channels (e.g., `categoryIDs`, `productIDs`).
*   **Ammunition Ready:** These IDs are now queued and ready. When Phase 4 unleashes the concurrent workers, they simply pop IDs from these Channels with near-zero latency, ensuring 100% accurate RPS and Latency benchmarks.

---

## PHASE 4: LOAD BLASTING & DYNAMIC PAYLOAD GENERATION (BLAST PHASE)
*(Handled by: `internal/runner/` and `internal/payload/`)*

*   **Worker Pool Initialization:** Spawns a pool of Goroutines based on the `load.concurrency` configuration (e.g., 100 concurrent threads). These workers continuously pull and execute Tasks from the Task Queue.
*   **Dynamic Fuzzing & Variable Chaining:**
    *   *Standard Fields:* Generates mutated/fuzzed data (excessively long strings, XSS payloads, negative integers) via the `payload.generator`.
    *   *Dependent Fields:* Workers randomly pop valid, pre-seeded IDs from the Go Channels populated in Phase 3.5.
*   **HTTP Execution:** Injects the Authorization Header and utilizes a highly optimized Connection Pool (`fasthttp` or a tuned `net/http` client) to flood the target server.
*   **Result Packaging:** Encapsulates the Status Code, Response Time, and Raw Body into a `Result` object and pushes it into the Result Queue (Non-blocking design).

---

## PHASE 5: ASYNCHRONOUS VALIDATION & AI HEURISTICS
*(Handled by: `internal/validator/` and `internal/ai/`)*

This pipeline runs asynchronously and parallel to the blast phase to prevent any degradation of the core RPS engine. A separate pool of Validator Workers continuously consumes the Result Queue.

*   **Hard Heuristic Engine (Default):**
    *   **Status Check:** Matches HTTP Status Codes (e.g., Expected 200, Actual 500 $\rightarrow$ FAIL).
    *   **Schema Check:** Parses the JSON Body and strictly validates it against the OpenAPI schema definitions.
*   **Soft Evaluation (AI Engine - Optional):**
    If the AI flag is enabled and the Hard Engine detects a complex anomaly (e.g., Status 200 but the body contains "Deadlock"):
    *   To optimize context windows and reduce token costs, **the AI is strictly fed a standardized payload**:
        *   `Request`: (Method, URL, Actual Payload)
        *   `Response`: (Status Code, Raw Body)
        *   `Schema`: (Expected format parsed from OpenAPI)
        *   `Expected`: (Expected behavior)
        *   `Context`: (Current Runtime State)
        *   `History`: (Recent error logs, if applicable)
    *   *Sample AI Payload Structure:*
        ```json
        {
          "Request": "POST /users",
          "Expected": 201,
          "Actual": 200,
          "Schema": "...",
          "Body": "..."
        }
        ```
*   **High-Throughput Logging:** All Results are immediately flushed to a temporary log file (JSONL) in preparation for report generation.

---

## PHASE 6: METRICS AGGREGATION, REPORTING & CLEANUP
*(Handled by: `internal/reporter/`)*

*   **Metrics Aggregation:** Calculates total requests, Pass/Fail ratios, and RPS (Requests Per Second).
*   **Latency Computation:** Implements a **High Dynamic Range (HDR) Histogram** with a pre-configured relative error tolerance (e.g., 1%). This is highly optimized for accurately calculating `P95` and `P99` latencies in high-performance computing environments.
*   **Live Dashboard:** Renders real-time charts and Progress Bars on the Terminal, utilizing UI throttling mechanisms to prevent CPU exhaustion.
*   **Report Generation:**
    *   Exports standard `JUnit XML` (Crucial for CI/CD pipelines like GitHub Actions/GitLab CI to parse failures).
    *   Renders a static, visually rich `HTML Report` for QA and Management review.
*   **Cache & State Update:**
    Instead of merely caching the OpenAPI hash, the `blaster/cached` directory persists crucial lifecycle metadata for subsequent differential runs. This ensures that even if the internal parser evolves, the runtime adapts seamlessly:
    *   `endpoint_yaml_hash`: Hash of the current endpoint configurations.
    *   `runtime_version`: Core engine version during the run.
    *   `payload_version`: Version of the Payload Generator module.
    *   `ai_version`: Version of the AI integration module.
    *   `report_version`: Version of the reporting structure.
*   **Exit Strategy (Exit Codes):**
    *   If any core test case fails or the error threshold is breached $\rightarrow$ Returns `os.Exit(1)` (Fails the CI/CD pipeline).
    *   If all tests pass or errors are within acceptable bounds $\rightarrow$ Returns `os.Exit(0)` (Greenlights the CI/CD pipeline for deployment).