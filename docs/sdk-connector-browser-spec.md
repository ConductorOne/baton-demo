# SDK connector browser

Status: proposed implementation specification. Written 2026-09-23. Implementation belongs in baton-sdk; this document is stored in baton-demo while the demo application viewer is developed independently.

## Purpose and product boundary

Provide a built-in local web interface for exploring a configured connector and invoking the operations it supports. An operator should be able to discover resource types, inspect live resources, entitlements and grants, and exercise provisioning without first producing a sync artifact or constructing protobuf requests manually.

Three distinct interfaces remain useful:

| Interface | Source | Purpose |
| --- | --- | --- |
| baton-demo application viewer | Live demo SQLite database | Observe the application changing after provisioning |
| SDK connector browser | Configured connector service methods | Inspect and exercise connector behavior |
| Existing baton explorer | A synced .c1z artifact | Analyze a captured inventory and expanded access |

The browser must identify itself as a live connector view. It does not imply that results represent a complete sync, a consistent global snapshot, or expanded effective access.

## Goals

- Ship inside connector binaries, with no Node.js runtime or separate frontend installation required by users.
- Use the existing connector factory, configuration, authentication and service contracts.
- Discover functionality from connector metadata, per-resource capabilities, annotations and operation schemas.
- Offer useful resource navigation and an advanced request/response console.
- Preserve pagination, identifiers, annotations and execution semantics faithfully.
- Make writes explicit, capability-aware and subject to server-side enablement.
- Support connectors using the existing builder adapters as well as custom service implementations.
- Keep operational behavior observable: pending requests, duration, errors, rate limits, partial results and uncertain write outcomes.

## Non-goals for the initial releases

- Hosted, multi-user administration, remote authentication or authorization policy management.
- Replacing ConductorOne orchestration, approval workflows, scheduled sync or audit storage.
- Automatically crawling all resources to build a graph, global search index or exact inventory totals.
- Implementing grant expansion or claiming to calculate effective access.
- A SQL editor, .c1z editor or connector configuration editor in the browser.
- Automatically exposing new RPCs merely because a protobuf descriptor exists.
- Loading arbitrary connector binaries into the initial implementation; it runs inside the connector process.

## Existing implementation anchors

Paths below are relative to baton-sdk and were inspected in the local checkout:

- `pkg/types/types.go`: `ConnectorServer` and `ConnectorClient` combine connector services.
- `pkg/connectorbuilder/connectorbuilder.go`: metadata and `GetCapabilities` derive support from registered implementations, including per-resource-type provisioning.
- `pkg/connectorbuilder/resource_provisioner.go`: Grant/Revoke dispatch, legacy adapters and existing retries.
- `pkg/cli/cli.go`, `pkg/cli/commands.go`: command registration, connector factories, runtime options and capability extraction.
- `proto/c1/connector/v2/`: protobuf services, request schemas, annotations and capability definitions.
- `pkg/baton/explorer/`: existing HTTP controller and .c1z-backed service.
- `frontend/`: React/TypeScript explorer and resource components.
- `cmd/baton/explorer.go`: embedded explorer command and local binding behavior.

The existing explorer backend uses store-specific queries and caches. Reuse suitable visual components, but do not route live calls through a fabricated .c1z store. A service method existing on `ConnectorServer` does not establish support: implementations can return `Unimplemented`.

## User workflows

### Start and inspect

Proposed command surface:

```sh
baton-example browse [existing connector configuration flags]
baton-example browse --provisioning [existing connector configuration flags]
```

Browser-specific settings: loopback port, optional browser launch, request timeout and maximum concurrent requests. Final flag names must be checked against SDK flags; in particular, avoid reusing `-p` for port when it denotes provisioning.

Startup loads configuration using the normal SDK path, creates the connector and required session context, validates it, loads metadata, starts a loopback listener, and prints the URL. A validation failure exits with the connector error. Metadata/capability incompleteness must be distinguishable from invalid credentials. Startup never starts a full sync or provisions anything.

The landing page shows connector display name, available resource types, supported operation categories, and whether writes are enabled. Configuration secrets never appear in this summary. Connection errors offer an explicit retry without silently creating duplicate connector instances.

### Navigate live data

1. Select a resource type and, where required, a parent resource.
2. Load one page of resources, retaining the response token and request context.
3. Open a resource to inspect traits, profiles, annotations, identifiers and parent links.
4. Load entitlements or grants for that resource, or use type-scoped listing where advertised.
5. Inspect the raw request and response alongside the formatted view.

Resource identity is the complete resource type/resource ID pair. Preserve external IDs, parent IDs and annotations; never recreate a provisioning object using only the display name or an assumed ID format. Unknown traits/annotations remain inspectable in raw form.

Search defaults to the loaded results and is labeled accordingly. There is no implicit exhaustive scan. Exact totals are omitted unless provided by a trustworthy source. Missing referenced resources are shown with their IDs and an unresolved state, rather than silently discarded.

### Grant and revoke

1. Select an entitlement and a principal of a type permitted by `grantable_to`.
2. Review target, principal, entitlement and request before submission.
3. Submit one explicit operation; disable duplicate submission while pending.
4. Show the response, including annotations and returned grants.
5. Re-read affected live data and show observation independently from operation success.

Revoke uses the actual selected grant object. Preserve all identifiers and annotations. Do not synthesize a grant ID. A Grant response with no grants can still be successful, including through legacy adapters. A timed-out mutation has an unknown outcome unless the service establishes otherwise; offer verification before retry.

### Advanced operations

A method console provides an operation selector, scoped target selection, schema-assisted inputs, an editable protobuf JSON request and the response. Friendly forms and raw requests share validation and dispatch. Raw mode cannot bypass write enablement or operation allowlisting.

Actions use advertised schemas and support invocation plus status polling. Account/resource creation, deletion, credential operations, ticket operations and events are introduced in separate increments. Asset streaming has a dedicated bounded download/preview path. Lifecycle methods such as Cleanup belong to the server lifecycle and are not ordinary user-invokable methods.

## Architecture

Proposed packages and ownership:

| Component | Responsibility |
| --- | --- |
| `pkg/connectorbrowser` | HTTP server, operation registry, capability resolution, request dispatch and lifecycle |
| SDK CLI integration | Configuration and connector factory integration; browser flags and shutdown |
| Frontend browser module | Live navigation, forms, method console and operation state |
| Shared frontend components | Reusable tables, details, trait rendering and JSON inspection |

The browser receives an initialized service plus runtime dependencies. HTTP handlers translate validated protobuf JSON into typed requests and invoke services in process. An external gRPC listener is not required. Where behavior currently lives in transport interceptors, extract or reuse the validation/session/error handling explicitly; direct invocation must not accidentally skip it.

Connector initialization and cleanup happen once per server lifetime. Request contexts carry cancellation, deadlines and required session state. Shutdown stops accepting requests, cancels outstanding work, waits within a bounded timeout and invokes cleanup exactly once. Streaming work has its own cancellation and bounded buffering.

Do not assume all connectors are concurrency-safe. Default to one active connector operation per instance, with bounded queuing and explicit cancellation. Raising concurrency is an opt-in implementation choice after compatibility verification. Avoid holding the execution slot while merely waiting between action status polls.

## Operation registry and capabilities

Use an explicit registry with an entry per supported operation:

- Stable operation ID and request/response protobuf descriptors.
- Typed invocation adapter.
- Read, mutation, streaming or lifecycle classification.
- Required connector/resource capability and scope requirements.
- Optional schema provider and friendly form renderer.
- Input validation, redaction and response-size policy.
- Timeout and cancellation behavior.

Descriptors describe data structure; the registry describes execution policy. New RPCs do not become accessible until classified and registered.

Resolve capabilities through existing SDK capability extraction where available, with metadata fallback consistent with the CLI. Combine resource-specific capabilities, annotations, static entitlement support, action schemas and optional service interfaces. Do not infer support by issuing a write. Legacy or incomplete capability information is shown as unknown; unsupported operations are unavailable and `Unimplemented` responses update the current session's view for that scope. Permission errors do not imply lack of implementation.

Provisioning support is evaluated for the entitlement's target resource type. Principal type restrictions are evaluated separately. Type-scoped entitlements/grants and static entitlements require explicit request construction using the SDK's annotated contracts; no assumed dummy resource is allowed.

## HTTP contract

Suggested endpoints, finalized during implementation:

| Endpoint | Behavior |
| --- | --- |
| `GET /api/session` | Sanitized connector metadata, operation availability, write policy and session nonce |
| `GET /api/operations` | Registered operation descriptions and request schemas |
| `POST /api/operations/{id}` | Validate and invoke a registered unary operation |
| Dedicated stream endpoints | Bounded assets/events where unary invocation is insufficient |

POST is appropriate even for read RPCs because their typed inputs may be structured. Read/mutation semantics come from the registry, never the HTTP verb alone. Responses carry protobuf JSON plus browser execution metadata: request ID, duration, operation ID and scope. Errors preserve gRPC status and relevant safe details; validation errors identify fields. Do not convert errors to successful empty lists.

Protobuf JSON encoding must use the protobuf serializer, preserving field presence, enum conventions, 64-bit values and `Any` handling. Unknown `Any` types must produce an explicit diagnostic or safe opaque representation, not lost data. Validate request bodies with size limits before dispatch. Treat all connector-provided display fields as untrusted text.

## Pagination, freshness and resource limits

- Fetch one page at a time; allow explicit next-page and bounded load-more actions.
- Keep opaque tokens intact and scoped to operation, resource type, parent, target and other request inputs.
- Changing scope clears the token chain and incompatible cached results.
- Preserve response annotations and session/sync context required by the contract; do not invent a full sync lifecycle for a browse request.
- Parent navigation follows advertised child types; cyclic references do not trigger recursive crawling.
- Label loaded results and last successful refresh. Stale data remains visibly stale on failure.
- Cache only within the session, with bounded size and explicit invalidation after writes.
- Default to manual refresh for real connectors; optional polling is bounded and pauses in hidden tabs.
- Honor existing SDK retry and rate-limit behavior. Do not layer automatic mutation retries over it.
- Surface rate-limit responses and backoff; prevent rapid repeated submissions.
- Apply deadlines, response limits and bounded request queues. An oversized result yields an explicit limit error, not an apparently complete truncated result.

## Local access and sensitive data

The initial browser binds only to loopback. Remote hosting requires a separate authentication/authorization design and is outside this spec's initial scope.

Use a per-process local browser session, strict Host and Origin validation, no permissive CORS, and a session nonce for RPC requests. Browser bootstrap must avoid persisting session tokens in logs or shareable URLs. Serve assets locally with a restrictive content security policy. Deny framing and avoid CDN dependencies.

All mutation adapters check write enablement on the server. Reuse provisioning semantics where applicable; later operation categories must have an explicit policy rather than inheriting enablement accidentally. No request parameter can turn writes on.

Connector credentials stay in the Go process. Request history is bounded, in memory, and redacted using schemas and operation-specific rules. Credential issuance and rotation results require deliberate reveal/copy, are excluded from general history/logs, and are cleared when dismissed. Arbitrary payloads may contain sensitive data, so raw request/response bodies are not logged by default. The UI cannot prevent an authorized operator from seeing source data returned by a read; its boundary is local access, not field-level authorization.

## Frontend behavior

- Navigation lists resource types using connector labels and generic fallbacks.
- Lists and detail views support keyboard navigation, focus management and accessible status announcements.
- A clear persistent indicator identifies live mode and write enablement.
- Empty, loading, failed, stale, unsupported and permission-denied states are distinct.
- Raw JSON is available without replacing the readable view.
- Write confirmation names the concrete operation and objects, with no ambiguous generic confirmation text.
- Polling never steals focus or closes an active form.
- Pending writes survive route changes in the current session's operation view; leaving the page does not assert cancellation succeeded upstream.
- Browser history preserves navigation context, but does not contain credentials, raw requests or secret results.

Reuse existing explorer components after separating their store-specific assumptions. Keep snapshot APIs working unchanged. The SDK browser must be buildable independently of a full baton command build, with embedded assets present in published modules and release artifacts.

## Delivery stages

### Stage 0: contracts and integration proof

Inventory all RPCs and their scope/capability mapping. Confirm CLI flag compatibility, connector/session setup, validation currently performed by transports, embedding/build strategy and resource-type annotations. Produce a bounded read-only proof using baton-demo and one differently structured connector. Record binary size impact. No public mutation endpoint yet.

### Stage 1: read-only browser

Deliver CLI startup, local session protection, capability display, resource types/resources, targeted retrieval where supported, entitlements, grants, pagination, raw JSON and explicit errors. Include nested resources and type-scoped/static listing coverage. No full crawl or global graph.

### Stage 2: Grant/Revoke

Deliver server-side provisioning gating, principal/entitlement selection, actual-grant revocation, confirmation, execution states, refresh and unknown-outcome handling. Test legacy empty-grant responses and delayed visibility. Use the independent baton-demo database viewer to observe effects.

### Stage 3: schemas and method console

Deliver registered unary operations incrementally: action schemas/invocation/status, resource and account management, ticket schemas/operations, and credential operations with dedicated secret handling. Include advanced JSON inputs and coverage that raw mode enforces identical policy.

### Stage 4: specialized services and polish

Add bounded asset streaming and event exploration, broader compatibility coverage, accessible interaction polish and documentation. Evaluate whether remote connector attachment or explicit snapshot export merits separate specifications.

## Verification and acceptance

Use the SDK review checklist to route each implementation increment by its actual failure mode. Mutation dispatch and credential handling need stronger review than static rendering. This proposal is not a correctness signoff or a frozen implementation plan.

Acceptance scenarios:

1. A freshly built connector starts its browser using normal configuration; no Node runtime, sync artifact or full sync is required.
2. Startup failure leaves no listener or abandoned connector resources; normal shutdown cleans up once.
3. A read-only connector exposes browse operations and no usable mutation path, including raw HTTP requests.
4. Capabilities differ correctly between resource types; absent, unknown and denied operations render differently.
5. Multi-page and nested-resource listings retain exact tokens/context; changing parents cannot reuse the previous parent's token.
6. Static and type-scoped entitlement/grant calls follow their actual contracts.
7. Grant passes the selected full principal and entitlement objects; Revoke passes the selected full grant.
8. A successful legacy Grant with no returned grants is shown as success and triggers verification.
9. A timed-out write is reported as uncertain and is never automatically resubmitted by the browser.
10. Post-write reads can remain stale without falsely reporting provisioning failure or immediate verified success.
11. Unknown annotations and unusual IDs, including punctuation and Unicode, survive round trips.
12. Rate limits, malformed input, oversized payloads, cancellation and stream termination produce bounded behavior and explicit errors.
13. Host/Origin/session checks reject cross-origin calls; read-only mode rejects all registered mutations server-side.
14. Credential inputs/results do not enter general logs, URL state or operation history.
15. The existing .c1z explorer remains functional after component extraction.

Verification layers: dispatcher/capability tests; HTTP contract tests against a fake service; integration tests against baton-demo; representative connector compatibility checks; browser tests for navigation, pagination, confirmations and stale/error states. Streaming and asynchronous operations require their own lifecycle tests when introduced.

## Decisions and remaining implementation questions

Decided: local embedded UI; direct connector service execution; read-only first; server-side write gates; explicit operation registry; existing protobuf contracts; no automatic global crawl; independent demo viewer.

Resolve during Stage 0: exact flags and defaults; shared frontend package boundaries; complete capability-to-operation mapping for older/custom connectors; transport validation/session requirements; conservative concurrency configuration; published asset strategy and size budget. Remote serving, persisted history and full inventory indexing each require a separate scope decision.
