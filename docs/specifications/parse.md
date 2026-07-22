# Parse Specification

## Overview

`blaster parse` transforms an OpenAPI specification into a complete Endpoint configuration.

OpenAPI always remains the Single Source of Truth (SSOT) for request schemas, response schemas, parameters, authentication, transport definitions, and API structure.

Parse analyzes OpenAPI together with any existing Endpoint configuration, discovers runtime identities and references, resolves deterministic mappings, optionally invokes AI for unresolved cases, validates every mapping, synchronizes Endpoint, and writes the final compiled Endpoint back to disk.

Planner never performs endpoint analysis again.

Planner only loads OpenAPI, Endpoint, and Workflow definitions.

This separation guarantees deterministic behavior, stable execution, reproducible planning, and a clear responsibility boundary between Parse and Planner.

---

# Glossary & Core Concepts

To maintain a ubiquitous language across the system, the following terms are strictly defined.

## Target Node

A Target Node is any data field identified as having identifier characteristics.

Examples include: id, Id, ID, userId, roleId, categoryId, uuid, UUID

A Target Node itself is neutral.

Its role depends entirely on where it appears.

---

## Identity (Producer)

An Identity is a Target Node located inside an API Response.
Identities may appear in: Response Body, Response Header.

It represents data produced by the backend and may later be consumed by other API operations.

Example:
```yaml
response:
  header:
    X-Resource-ID: identity
  body:
    id: identity
    organizationId: identit
```
A single operation may export multiple identities.

---

## Relative (Consumer)

A Relative is a Target Node located inside an API Request.

Relatives may appear in: Path, Query, Header, Cookie, Request Body

A Relative consumes an Identity exported by another operation.

Example

```yaml
body:

  ownerId: GET:users:id
```

Multiple Relatives may exist within the same operation.

Example

```text
GET /groups/{groupId}/users/{userId}/permissions
```

becomes

```yaml
path:

  groupId: GET:groups:id

  userId: GET:users:id
```

Each Relative is resolved independently.

---

## Mapping

A Mapping connects one Relative to one exported Identity.

Mappings always point to a valid Identity.

Example

```yaml
body:

  ownerId: GET:users:id
```

---

# Design Philosophy

Parse is the only component responsible for endpoint analysis.

Planner is never responsible for discovering identities, resolving references, performing semantic analysis, fuzzy matching, AI reasoning, or modifying Endpoint configuration.

Every runtime decision required for workflow generation must be completed during Parse.

Endpoint therefore represents the compiled analysis result produced by Parse.

Developer modifications remain first-class citizens and are preserved whenever they remain compatible with the current OpenAPI specification.

---

# Responsibilities

Parse performs the following responsibilities in order.

1. Load OpenAPI.
2. Load existing Endpoint.
3. Discover Identities.
4. Discover Relatives.
5. Build the Identity Index.
6. Resolve deterministic mappings.
7. Validate mapping compatibility.
8. Collect unresolved references.
9. Optionally invoke AI.
10. Synchronize Endpoint.
11. Write Endpoint back to disk.

Planner never repeats any of these responsibilities.

---

# Endpoint Compilation

Endpoint is the complete compiled metadata produced by Parse.

Every discovered Identity and every discovered Relative is explicitly written into Endpoint regardless of whether it was resolved automatically, resolved manually, resolved by AI, or remains unresolved.

Nothing is inferred later.

Everything Parse understands is explicitly recorded.

Example

```yaml
response:

  id: identity

body:

  userId: GET:users:id

  ownerId: ?
```

---

# Endpoint Layout

The generated Endpoint directory always mirrors the original OpenAPI route hierarchy.

Parser never reorganizes, renames, merges, flattens, or restructures API routes.

Example

```text
GET  /users
POST /users/login
PUT  /users/{id}
GET  /users/{id}/profile
```

becomes

```text
endpoint/

users/
    get.yaml

    login/
        post.yaml

    {id}/
        put.yaml

        profile/
            get.yaml
```

Every Endpoint file can always be located directly from its corresponding OpenAPI operation.

---

# Producer–Consumer Model

The system enforces a strict one-way data flow.

Requests never export identities.

Even if a Request contains an identifier, it is still considered a Relative because it consumes existing data.

Only Responses export identities.

True identities are generated and validated by the backend and therefore originate exclusively from Response schemas.

---

# Identity Discovery

Parser discovers Identities recursively throughout every Response schema, explicitly including both Response Bodies and Response Headers.

Parser discovers Identities recursively throughout every Response schema.

Discovery follows deterministic rules only.

Priority order is:

1. Existing Identity definitions already present in Endpoint.
2. Field names starting with or ending with an identifier token such as `id` or `uuid` (case-insensitive), provided that:
   - the field type is `string`, `integer`, or `number`; and
   - the identifier token is explicitly distinguishable from the adjacent characters. Otherwise, the field remains undetermined.
3. OpenAPI schema format or type explicitly representing an identifier (for example `uuid`).
4. Continue recursively through every nested object and array.
5. Export every discovered Identity.

Parser intentionally does not infer business-specific identifiers such as `code`, `slug`, `number`, `token`, or similar fields unless explicitly configured by the developer or AI.

---

# Relative Discovery

Parser discovers every Relative recursively from every request location.

Discovery includes Path parameters, Query parameters, Header parameters, Cookie parameters, Request Body fields, and nested request objects.

Every discovered Relative is explicitly written into Endpoint.

No request reference is hidden.

---

# Identity Index

After discovery completes, Parse builds an in-memory Identity Index.

Only operations exporting at least one Identity participate.

Each Identity entry stores:

- HTTP Method
- Resource
- Identity Name
- OpenAPI Schema Type

The Identity Index exists only during Parse and is never written to disk.

---

# Deterministic Resolution

Parser resolves every Relative using deterministic rules only.

Parser never performs semantic reasoning, business inference, natural language understanding, or AI reasoning during deterministic resolution.

Resolution follows an Inside-Out strategy.

Parser first searches nearby operations sharing the same route hierarchy or OpenAPI Tag.

If no suitable candidate exists, Parser expands the search to the global Identity Index.

Every candidate receives a deterministic similarity score.

A mapping is accepted only when all of the following conditions are satisfied:

- the best candidate score exceeds the configured minimum threshold;
- the difference between the best candidate and the second-best candidate exceeds the configured safety margin;
- the Identity schema type is compatible with the Relative schema type.

Otherwise the Relative remains unresolved.

Parser never guesses ambiguous mappings.

---

# Pending References

Whenever deterministic resolution cannot determine exactly one valid Identity, the Relative becomes Pending.

Pending references are collected into an internal Pending Basket.

After deterministic analysis completes, the Pending Basket becomes the input for optional AI analysis.

Pending references that remain unresolved are explicitly written into Endpoint.

Example

```yaml
body:

  ownerId: ?
```

Pending references are never silently ignored.

They remain visible for manual configuration or future AI analysis.

---

# Explicit Ignore Directive

Not every Target Node containing an identifier token acts as an Identity or Relative. Developers may explicitly reject incorrect deterministic classifications using the `ignore` directive.

When a developer sets a node to `ignore` in the Endpoint configuration, Parse permanently excludes that node from all future analysis, discovery, and synchronization cycles.

Example:
```yaml
response:
  id: identity
  transactionABCid: ignore

body:
  userId: GET:users:id
  trackingID: ignore
The ignore directive acts as a hard boundary. Parse respects this directive explicitly and never overwrites it, ensuring that developer overrides remain permanent across OpenAPI version bumps.
```

---

# AI Analysis

AI analysis is optional.

AI is invoked only after deterministic resolution has completed.

Parse never sends the entire OpenAPI specification.

Instead, Parse constructs a compact analysis payload containing:

- the current API operation;
- discovered Identities;
- discovered Relatives;
- the Identity Index;
- the Pending Basket;
- minimal surrounding API context.

AI has only three responsibilities.

First, review Pending Identity candidates and determine whether they should become exported Identities.

Second, discover additional Identities that deterministic discovery intentionally does not infer.

Third, connect Pending Relatives to existing Identities.


AI never modifies OpenAPI.

AI never changes routing.

AI never changes schemas.

AI only generates Endpoint mappings.

Mappings that remain unresolved after AI analysis continue to be written as Pending.

---

# Type Validation & Soft Coercion

Successful name matching alone is insufficient; Parser must validate schema compatibility using OpenAPI.

However, Parser acknowledges HTTP transport constraints where primitive types frequently overlap. Parser enforces Soft Coercion rules for identifiers.

A mapping is accepted if the schema types are strictly identical OR if they belong to a compatible coercion group:
- `string` and `integer` are inherently mutually compatible for identifiers.

Example of Accepted Mapping:
```text
Identity (Response)
id : integer

Relative (Request)
userId : string
```
If an Identity and a Relative expose fundamentally incompatible types (e.g., object vs boolean), the mapping is rejected and becomes Pending.

---

# Endpoint Synchronization

Before analysis begins, Parse loads the existing Endpoint configuration.

Existing Endpoint definitions are treated as authoritative whenever they remain compatible with the current OpenAPI specification.

During synchronization Parse may:

- preserve valid developer mappings;
- update deterministic mappings;
- create newly discovered mappings;
- remove mappings whose producer no longer exists;
- validate every existing mapping.

Mappings referencing missing producers become Pending.

Removed OpenAPI operations remove their corresponding Endpoint files.

Endpoint always reflects the latest compatible state of the current OpenAPI specification.

---

# Planner Contract

Planner never analyzes Endpoint.

Planner never discovers Identities.

Planner never discovers Relatives.

Planner never performs deterministic matching.

Planner never performs fuzzy matching.

Planner never invokes AI.

Planner only loads:

- OpenAPI;
- Endpoint;
- Workflow.

Planner builds the execution plan entirely from the compiled metadata produced by Parse.

---

# Runtime Contract

Runtime never reads Endpoint directly.

Runtime executes only the compiled execution plan generated by Planner.

All identity discovery, reference discovery, deterministic resolution, AI analysis, validation, synchronization, and endpoint compilation are completed before execution begins.

Runtime therefore contains no endpoint intelligence and performs no dynamic inference during execution.