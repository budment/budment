# Endpoint Specification

## Purpose

The Endpoint domain is the core semantic engine of Blaster. It defines how API metadata is modeled, how data flows between endpoints (Producers and Consumers), and how human-driven decisions (Declarative State) are prioritized over automated algorithms.

This specification outlines the mental model, definitions, and strict rules used throughout the parsing engine. It is strictly **Protocol-Agnostic**—meaning it manages relationships across REST, gRPC, GraphQL, and event streams seamlessly.

---

## Target Nodes

A Target Node is a schema field that satisfies the conditions to participate in data flow mapping.

A Target Node is **NOT** restricted to just `id` or `uuid`. It can be **ANY** field that acts as a Producer or Consumer in the API schema (e.g., `url`, `code`, `slug`, `invoiceNumber`, `receipt_handle`).

When a Target Node in a Request (Consumer) is mapped to a Target Node in a Response (Producer), it means the system must use the exact real data originating from the database (e.g., a real response `url`), rather than randomly generating fake data.

**Rule of Mapping:** One Target Node strictly maps to exactly **ONE** other Target Node.

---

## Identity (Producer)

An Identity represents a resource produced by the API (e.g., `User.id`, `Product.url`, `Order.code`).

An Endpoint **MAY** declare multiple Identities.

Each Identity has exactly **ONE** Root.

---

## Branch Identity

A Branch Identity is an alias of a Root Identity. Both represent the exact same resource.

Branch Identities exist to support APIs that expose the same resource through different operations.

```text
Root
 └── Branch
```

### Example

`POST /login` produces `User.id`

`GET /me` produces `User.id`

Both produce the same User resource. One is chosen as the Root, and the other becomes a Branch pointing to that Root.

---

## Relative (Consumer)

A Relative references an Identity. It represents data the client must provide in the Request.

Resolution strictly follows the Identity tree downward.

Given the following Identity Tree:

```text
A
 ├── B
 └── C
```

If a Relative maps to **A**, it accepts data produced by: **A, B, C**.

If a Relative maps to **B**, it accepts data produced by: **B, C**.

If a Relative maps to **C**, it accepts data produced by: **C**.

Matching is strictly directional. A child never resolves to a parent.

---

## Branch Depth

The Endpoint conceptual specification supports arbitrary Identity depth:

```text
A
 └── B
     └── C
         └── D
```

However, the current Blaster Parse implementation generates only one Branch level automatically:

```text
A
 └── B
```

This keeps automatic discovery deterministic and performant while successfully covering the vast majority of real-world APIs. Developers can manually create deeper Branch hierarchies in the YAML files if required.

---

## Protocol Agnosticism & Cross-Protocol Mappings

The Endpoint layer is the semantic bridge between API descriptions (OpenAPI, gRPC, GraphQL, Kafka) and Workflow Planning. It never contains execution logic.

### Directory Structure

Every protocol owns its directory namespace:

```text
endpoint/
    rest/
    grpc/
    graphql/
    kafka/
```

### REST Priority

REST is considered the canonical protocol whenever it exists.

- REST Roots cannot reference identities from other protocols.
- REST Branches can only reference REST Roots.
- Other protocols may reference REST Roots.
- Other protocols may create independent Roots when REST does not provide them.

This guarantees stable identity ownership while allowing cross-protocol execution (e.g., A Kafka Event providing a TargetID for a gRPC Request).

---

## Addressing (Absolute vs. Relative)

To provide the best Developer Experience (DX), Blaster YAML files support Relative Addressing for intra-protocol relationships.

### 1. Absolute Addressing (Memory & Cross-Protocol)

Contains the protocol prefix. Required when mapping across different protocols.

**Format:**

```text
{protocol}:{method}:{path}:{field}
```

**Example:**

```text
rest:GET:/users:id
grpc:UserService/Get:id
```

### 2. Relative Addressing (Intra-Protocol)

Omits the protocol prefix. The parser automatically infers the protocol based on the file's location.

**Format:**

```text
{method}:{path}:{field}
```

**Example:**

```text
GET:/users:id
```

(inside a `rest/` directory).

The Parser Engine automatically converts Relative to Absolute addresses during in-memory processing, and strips redundant prefixes when writing back to disk.

---

## Pending & Ignore States

### Pending (?)

Pending indicates that no Root Identity could be resolved automatically.

```yaml
invoiceNumber: ?
```

Pending Target Nodes require manual confirmation by a human developer or resolution via optional AI analysis. They never resolve automatically on their own.

### Ignore (`ignore`)

Ignored Target Nodes participate in State Reconciliation (to preserve the developer's choice) but are completely excluded from Discovery and Resolution algorithms. Ignore affects discovery only.

---

## Core Rules Summary

| Rule | Requirement |
|------|-------------|
| One Root | Every Identity belongs to exactly one Root. |
| Branch | A Branch represents the exact same resource as its Root. |
| Relative | A Relative resolves downward only (Descendants are compatible, Parents are not). |
| Pending | Never resolves automatically. Requires manual or AI intervention. |
| Ignore | Never participates in automatic discovery or resolution. |
| Cross-Protocol | Identifiers can seamlessly map across REST, gRPC, GraphQL, etc. |