# Blaster Parse Lifecycle Specification

The parsing phase is responsible for transforming an OpenAPI specification into a deterministic endpoint configuration that the runtime can execute without requiring additional reasoning.

Unlike traditional API testing tools that simply import OpenAPI into collections or generate request templates, Blaster synchronizes the current OpenAPI specification with the project's endpoint configuration, preserves user-defined metadata while the underlying OpenAPI schema remains unchanged, and automatically enriches newly discovered fields with semantic information and runtime strategies.

The endpoint configuration is considered a project asset rather than a temporary cache. Developers are encouraged to review, customize, version-control, and continuously maintain it as the API evolves.

Runtime execution never invokes AI or performs additional semantic analysis. Every execution decision required by blaster run has already been prepared during the parsing phase.

## Phase 1 — Load OpenAPI & Endpoint Configuration

Blaster begins by loading the configured OpenAPI specification together with the existing endpoint configuration defined by the project.

The OpenAPI specification is always treated as the single source of truth describing the API surface, Endpoint configuration never defines API structure. It only enriches the structure already defined by OpenAPI.

If no endpoint configuration exists, Blaster starts with an empty configuration and generates everything from the OpenAPI specification.

## Phase 2 — Synchronize Endpoint Configuration

Rather than regenerating every endpoint from scratch, Blaster synchronizes the existing endpoint configuration with the latest OpenAPI specification.

For every endpoint and every schema field, Blaster compares the current OpenAPI definition with the existing endpoint metadata and applies deterministic synchronization rules.

If an endpoint or field already exists and both the field name and schema type remain unchanged, Blaster preserves the entire existing configuration without modification. User-defined semantic rules, validators, payload strategies, planner hints, and every manual customization remain untouched.

If OpenAPI introduces a completely new endpoint or new schema field that does not yet exist inside the endpoint configuration, Blaster automatically creates a new endpoint entry using the default runtime configuration. Additional semantic enrichment may be performed later if Advanced Parsing is enabled.

If an endpoint or schema field no longer exists inside the OpenAPI specification, the corresponding endpoint metadata is automatically removed because it no longer represents a valid part of the current API.

If an existing field still exists but its schema changes, such as a different primitive type, object structure, array definition, or any other structural change, Blaster considers the previous metadata obsolete. All runtime metadata attached to that field, including semantic definitions, validation strategies, payload generation rules, planner hints, and any automatically generated configuration, is discarded. The new schema from OpenAPI becomes the new baseline before semantic enrichment begins again.

This synchronization process guarantees that endpoint metadata always remains structurally consistent with the latest OpenAPI specification while preserving every valid manual configuration created by developers.

## Phase 3 — Semantic Enrichment

After synchronization, Blaster processes only the newly created fields or fields whose previous metadata became obsolete due to schema changes. Existing fields whose schema remains unchanged are skipped entirely.

If Advanced Parsing is disabled, Blaster simply assigns the default runtime strategy based on each primitive type. For example, integers use the built-in integer strategy, strings use the default string strategy, arrays use the default array strategy, and so on. No semantic inference is performed.

If Advanced Parsing is enabled, every affected endpoint is analyzed independently using the user's configured AI provider.

The AI is never responsible for generating test cases, assertions, business logic, execution flow, or runtime behavior.

Instead, the AI receives a standardized prompt containing the endpoint definition together with Blaster's built-in semantic library and any project-defined hybrid semantic extensions.

The semantic library contains reusable concepts already supported by the runtime, including email, phone, UUID, URL, IP address, username, password, quantity, stock, price, currency, address, postal code, timestamp, date, identifier, and many other common API concepts.

The AI simply determines whether a field matches one of these predefined semantic categories and attaches the corresponding runtime semantic if appropriate.

For example, fields named email, mail, correo, gmail, or equivalent names in other languages may all be mapped to the built-in Email 

For numerical fields such as stock, quantity, inventory, or price, the AI may determine that the default integer strategy should be narrowed into a more specific positive-number constraint based on semantic understanding rather than simply relying on primitive data types.

The AI never invents new runtime rules. It only selects semantic capabilities that already exist inside Blaster or inside the project's hybrid semantic definitions.

Whenever the AI cannot confidently determine additional semantics, it simply leaves the field unchanged. Runtime automatically falls back to the default testing strategy for the underlying primitive type.

## Additional — Hybrid Semantic Extension

Developers may extend Blaster's built-in semantic library by defining additional semantic types inside the project configuration.

Hybrid semantics allow teams to introduce organization-specific concepts without modifying Blaster itself.

These custom semantic definitions become part of the parsing context and are provided to the AI together with the built-in semantic library.

As a result, the AI can recognize organization-specific field names exactly the same way it recognizes built-in semantic types.

Final Output

After parsing completes, the project contains a deterministic endpoint configuration fully synchronized with the latest OpenAPI specification.

Every endpoint is guaranteed to match the current API structure, obsolete metadata has been removed, newly introduced endpoints have received executable runtime configuration, existing manual configurations have been preserved whenever still valid, and optional semantic enrichment has already been completed.

The synchronized endpoint configuration together with the latest OpenAPI specification become the execution source for blaster run, allowing the runtime to execute tests deterministically without performing additional semantic analysis or AI reasoning.

