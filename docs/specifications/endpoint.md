# Endpoint Specification

## Overview

Endpoint defines runtime identity mappings that cannot be determined reliably from OpenAPI alone.

OpenAPI always remains the Single Source of Truth (SSOT) for request schemas, response schemas, parameters, authentication, transport definitions, and API structure.

Endpoint never duplicates OpenAPI. It only supplements identity information required for deterministic planning.

Planner loads OpenAPI together with Endpoint before generating workflows and execution plans.

Runtime never reads, modifies, or analyzes Endpoint configuration.

---

## Glossary & Core Concepts

To maintain a ubiquitous language across the system, the following terms are strictly defined.

### Target Node

A data field identified as having identifier characteristics.

Examples include: id, Id, ID, uuid, UUID, userId, categoryId, ownerId

A Target Node itself is neutral.

Its role depends entirely on where it appears.

### Identity (Producer)

An Identity is a Target Node located inside an API Response.

It represents data produced by the backend and may later be reused by other operations.

Example

```yaml
response:

  id: identity

  userId: identity
```

### Relative (Consumer)

A Relative is a Target Node located inside an API Request.

It consumes an Identity previously exported by another operation.

Relatives may appear in: Path, Query, Header, Cookie, Request Body

Example

```yaml
body:

  ownerId: GET:/users:id
```

### Mapping

A Mapping connects one Relative to one exported Identity.

Mappings always point to an exported Identity.

---

## Design Philosophy

Endpoint is intentionally minimal.

It does not describe: business logic, validation rules, payload generation, workflow definitions, execution order, planner hints, semantic meaning, runtime behavior, API documentation

Its only responsibility is defining runtime identity relationships that cannot be inferred safely from OpenAPI alone.

Whenever deterministic conventions are sufficient, Endpoint should not be generated.

---

## Parse (`blaster parse`)

`blaster parse` analyzes an OpenAPI specification and generates the Endpoint directory.

The generated directory always mirrors the original OpenAPI route hierarchy.

Parser never reorganizes, renames, merges, or flattens API routes.

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

## Producer–Consumer Model

The system enforces a strict one-way data flow.

Requests never export identities.

Even if a Request contains an `id`, it is still considered a Relative because it consumes data.

Only Responses export identities.

True identities are generated and validated by the backend, therefore they originate exclusively from Response schemas.

---

## Identity Discovery

Parser discovers exported Identities using deterministic rules only.

Discovery priority:

1. Explicitly marked by Developer or AI.
2. Response field named `id`.
3. Response field named `uuid`.
4. Recursive traversal continues until every nested object has been inspected.

Parser intentionally does **not** guess business-specific identifiers such as: code, slug, number, token, key

unless they are explicitly configured.

A single Response may export multiple identities.

Example

```yaml
response:

  id: identity

  organizationId: identity
```

---

## Identity Index

After discovery, Parser builds an in-memory Identity Index.

Only operations exporting at least one Identity participate.

Dynamic parameter folders are ignored.

Examples

```text
{id}
{userId}
{categoryId}
```

do not become independent resources.

The Identity Index stores:

- resource path
- operation
- exported identities

Example

```text
users

    GET
        id

products

    GET
        id

categories

    GET
        id

users/me

    GET
        userId
```

The Identity Index exists only during planning and is never written to disk.

---

## Deterministic Resolution

Parser resolves every Relative using deterministic rules only.

Parser never performs:

- semantic inference
- natural language understanding
- business reasoning
- AI reasoning

Resolution follows an "inside-out" search strategy.

Parser first searches nearby resources sharing the same route hierarchy or OpenAPI context.

If no valid mapping is found, Parser expands the search to the global Identity Index.

A mapping is accepted only when exactly one candidate satisfies the configured confidence threshold.

If multiple candidates satisfy the threshold, the mapping remains unresolved.

Parser never guesses.

---

## Pending References

If deterministic resolution cannot determine a unique Identity, the Relative becomes Pending.

Pending references are written into Endpoint.

Example

```yaml
body:

  ownerId: ?
```

A Pending reference indicates:

- deterministic resolution failed
- no valid manual override exists
- user intervention may be required

Parser never invents mappings for Pending references.

---

## Minimal Endpoint Generation

Endpoint is generated only when manual information is required.

Parser does not generate Endpoint files for operations that are already completely understood.

For example,

```
GET /users
```

whose Response exports `id`, and

```
GET /users/{id}
```

whose Path clearly consumes that identity,

produce no Endpoint configuration because everything can be resolved deterministically.

Endpoint files are generated only when:

- deterministic resolution fails
- a manual override exists
- AI proposes additional mappings
- developer intentionally customizes behavior

The objective is to keep the Endpoint directory as small and maintainable as possible.

---

## Manual Configuration

Developers may manually override any automatic decision.

Example

```yaml
body:

  ownerId: GET:/users:id
```

or

```yaml
response:

  organizationId: identity
```

Manual configuration always has higher priority than deterministic rules.

Future parses preserve manual mappings whenever the corresponding OpenAPI operation still exists.

---

## AI Analysis

AI analysis is optional.

AI is invoked only after deterministic resolution finishes.

Instead of receiving the entire OpenAPI specification, AI receives a compact payload containing only the required context.

The payload consists of:

- the current operation
- exported identities already confirmed by Parser
- unresolved Pending references
- minimal surrounding API context

AI has two responsibilities.

### Connect

Resolve Pending references.

Example

```yaml
body:

  ownerId: ?
```

↓

```yaml
body:

  ownerId: GET:/users:id
```

### Discover

Identify additional exported identities missed by deterministic discovery.

Example

```yaml
response:

  orderCode
```

↓

```yaml
response:

  orderCode: identity
```

AI never modifies OpenAPI.

AI never changes API structure.

AI only proposes editable Endpoint configuration.

---

## Existing Endpoint Synchronization

Parser always loads both OpenAPI and the existing Endpoint directory.

Existing Endpoint is treated as the authoritative source for all developer-authored configuration.

When OpenAPI changes, Parser synchronizes only the affected operations.

Parser may:

- add newly required mappings
- remove mappings whose referenced OpenAPI operation no longer exists
- update generated information

Parser must never overwrite valid manual configuration.

Parser must never remove developer-authored mappings automatically unless the underlying OpenAPI operation has disappeared.

---

## Runtime

Endpoint participates only during planning.

Planner combines:

- OpenAPI
- Endpoint
- Identity Index

to build a deterministic Identity Graph.

Workflow generation consumes the compiled graph.

Runtime consumes only the compiled execution plan.

Runtime never performs:

- identity discovery
- endpoint analysis
- automatic mapping
- AI inference
- semantic reasoning

All identity resolution is completed before execution begins.