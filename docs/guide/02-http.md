---
title: HTTP Requests, Mutations & Responses
description: Guide to configuring HTTP methods, mutation pipelines, response assertions, and payload handling within Budment's two-phase execution architecture.
---

# HTTP Requests, Mutations & Responses

Budment's HTTP client is tightly integrated into its native Go execution engine, managing persistent connection pools, high-throughput I/O, and automated metric extraction without burdening the JavaScript runtime.

## 1. HTTP Methods & Target URLs

Initialize request builders using the `http` factory. These builders must be exported within a scenario pipeline to be executed by the engine.

```typescript
import { http } from '@budment/sdk';

export default [
    http.get("https://api.example.com/v1/users"),
    http.post("https://api.example.com/v1/users"),
    http.put("https://api.example.com/v1/users/42"),
    http.delete("https://api.example.com/v1/users/42")
];
```

Target URLs support dynamic expressions using SDK template strings, which are resolved natively by the Go engine during execution:

```typescript
import { http, env, get } from '@budment/sdk';

export default [
    http.get(`https://${env('API_HOST','api.example.com')}/users/${get('user_id')}`)
];
```

## 2. Pre-Request Pipeline 

The `.before()` method is not just a single configuration object; it is a **sequential execution pipeline**. It accepts variadic arguments (`...args`), allowing you to chain declarative objects, dynamic JavaScript hooks, and operational nodes (like logs or barriers) in a strict, predictable order.

```typescript
import { http, log, barrier } from '@budment/sdk';

http.post("https://api.example.com/orders")
    .before(
        // 1. JS Hook executes first
        (req) => { req.set({ ts: Date.now() }); },
        // 2. Logs output to the console
        log("Order request initialized"),
        // 3. Workers wait here until the quorum is met before firing the HTTP request
        barrier("sync_orders", { quorum: 50 })
    );
```

### Declarative Mutations (Static)

For static configurations, pass a declarative object containing `headers` and `body`. The SDK automatically serializes plain JavaScript objects into JSON during the compilation phase.

```typescript
import { http, env, get, random } from '@budment/sdk';

http.post("https://api.example.com/orders")
    .before({
        headers: {
            "Content-Type": "application/json",
            "Authorization": `Bearer ${env('AUTH_TOKEN')}`
        },
        body: {
            order_id: random.uuid(),
            quantity: 2,
            customer_ref: get("customer_id")
        }
    });
```

### Dynamic Script Hooks

When payloads require cryptographic signatures, query parameter mutations, or conditional logic, provide a callback receiving the `HttpRequest` instance:

```typescript
http.post("https://api.example.com/secure-data")
    .before(req => {
        const nonce = Date.now().toString();
        
        req.set(
            { payload: "payload_value", nonce }, // Body
            {
                headers: { "X-Request-Nonce": nonce },
                params: { format: "compressed" },      // Mutates URL query string
                path: "/v2/secure-data"                // Overrides request path
            }
        );
    });
```

### The HttpRequest Interface

```typescript
export interface RequestOptions {
    headers?: Record<string, any>;
    path?: string;
    params?: Record<string, any>;
}

export interface HttpRequest {
    readonly url: string;
    readonly method: string;
    
    // Updates outgoing body and options (headers, path, query params).
    // Objects are automatically serialized to JSON.
    set(body: object | string | ArrayBuffer | Uint8Array, options?: RequestOptions): void;
    
    // Wraps raw file data into a multipart descriptor
    file(data: ArrayBuffer | Uint8Array | string, filename?: string, contentType?: string): FileData;
    
    // Queries request body using GJSON selectors
    json<T any>(selector?: string): T | undefined;
}
```

## 3. Response Pipeline

Similar to `.before()`, the `.after()` pipeline sequentially processes response codes, asserts integrity, and extracts state.

### Declarative Assertions

Extract JSON fields natively without waking up a JavaScript VM, storing them directly into the Virtual User's memory scope:

```typescript
http.post("https://api.example.com/auth/login")
    .after({
        expect: {
            status: 200,
            bodyContains: "access_token"
        },
        extract: {
            // Extracts data.token and stores it as 'jwt_token' in VU context
            "data.token": "jwt_token"
        }
    });
```

### Dynamic Response Hooks

Use a JavaScript callback when validations require complex branching. Calling `res.json()` without arguments parses the entire payload into a JavaScript object:

```typescript
import { http, set, abort } from '@budment/sdk';

http.get("https://api.example.com/account/profile")
    .after((res, req) => {
        // 1. Query specific field using GJSON syntax
        const balance = res.json<number>("account.current_balance");
        
        // 2. Or parse the entire response body
        const fullProfile = res.json(); 

        if (balance === undefined || balance < 0) {
            abort("Invalid account balance detected. Halting iteration.");
        }

        set("user_profile", fullProfile);
    });
```

### The HttpResponse Interface

```typescript
export interface HttpResponse {
    readonly status: number;          // HTTP status code (0 for socket/dial errors)
    readonly headers: Record<string, string>;
    readonly error?: string;          // Network timeout or socket error description
    
    // Extracts via GJSON, or parses full body if no selector is provided
    json<T any>(selector?: string): T | undefined; 
}
```

## 4. Multipart Form Data & File Uploads

Uploading binary assets (images, PDFs, archives) requires preserving raw byte streams. Use `open(path, 'b')` to read the file into an `ArrayBuffer`, then attach it using `req.file()`:

```typescript
import { http, open } from '@budment/sdk';

// Read file into an ArrayBuffer during compilation
const invoiceData = open("./fixtures/invoice.pdf", "b");

export default [
    http.post("https://api.example.com/documents/upload")
        .before(req => {
            req.set({
                category: "finance",
                document: req.file(invoiceData, "invoice.pdf", "application/pdf")
            });
        })
        .after({
            expect: { status: 201 },
            extract: { "document.id": "uploaded_doc_id" }
        })
];
```

When `req.set()` receives fields containing `req.file()` descriptors, the Go engine automatically formats the payload as `multipart/form-data` and injects the corresponding `Content-Type` boundary header.

## 5. Error & Failure Classifications

Budment strictly distinguishes between business assertion failures, standard HTTP errors, and physical network errors. These define whether a request is flagged as successful (`IsSuccess`) in the engine's SLA metrics:

| **Classification** | **Trigger Condition** | **Engine Behavior & Metric Impact** |
| --- | --- | --- |
| **HTTP Status Error** | Server returns `Code < 200` or `Code >= 400`. | Automatically marks `IsSuccess = false`. Reflected in standard `http_req_failed` rates. |
| **Assertion Failure** | Fails a declarative `expect` condition. | Automatically marks `IsSuccess = false`. |
| **Network/I/O Error** | Socket timeouts, DNS failures, connection resets. | Returns `Status: 0`, sets `ErrorMessage`. Marks `IsSuccess = false`. |
| **Logic Failure** | Manual call to `fail(reason)` in JS hook. | Logs a `FAIL` event and continues executing the pipeline. |
| **Iteration Abort** | Manual call to `abort(reason)` in JS hook. | Immediately terminates the JS hook, skips remaining pipeline steps, and restarts the VU iteration. |