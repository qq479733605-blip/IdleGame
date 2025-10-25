# IdleMMO Server Refactor Plan

This plan captures the high-level direction for evolving the IdleMMO server into an
extensible, production-ready backend by applying modern game-server design patterns.
It documents the architectural target state, the module layout introduced in this
iteration, and recommended next steps for continued refactoring.

## Vision

* Embrace a **modular runtime** where vertical features can be plugged in and removed
  without touching core bootstrapping code.
* Keep gameplay rules and progression logic isolated inside dedicated **domain/core
  packages** so that infrastructure concerns (networking, storage, scheduling) remain
  replaceable.
* Standardize on **actor-driven orchestration** for real-time features while exposing
  synchronous application services for web / API workflows.
* Make it easy to swap persistence implementations (JSON, SQL, cloud storage) by
  leaning on repository interfaces and dependency inversion.

## Runtime Architecture

The `internal/app/runtime` package now provides a lightweight application framework:

| Component | Responsibility | Patterns |
|-----------|----------------|----------|
| `App` | Orchestrates module lifecycle (configure → start → stop) and ensures reverse-order shutdown. | Template Method, Module Manager |
| `Container` | Service registry used for dependency inversion between modules. | Service Locator, Dependency Injection |
| `Module` interface | Contract that every feature module implements. | Plugin Architecture |

### Registered Modules

| Module | Location | Purpose | Key Services Provided |
|--------|----------|---------|-----------------------|
| Config | `internal/app/modules/config` | Loads static domain/equipment data at boot. | — |
| Actors | `internal/app/modules/actors` | Spins up the Proto.Actor system, scheduler, persistence, and gateway actors. | Actor system, root context, actor PIDs, game repository |
| User | `internal/app/modules/user` | Supplies the registration service, JSON-backed repository, and HTTP handler. | User repository, registration service, Gin handler |
| HTTP Transport | `internal/app/modules/transport/http` | Hosts Gin, applies cross-cutting middleware, and wires HTTP/WebSocket routes. | Gin engine, HTTP server |

Each module only depends on the runtime container. This isolates code paths and enables
feature teams to ship new functionality (e.g., guilds, leaderboards) as independent
modules without editing `main.go` beyond a single `app.Register` call.

## Extensibility Patterns

1. **Ports & Adapters**: Core business logic lives under `internal/core`. Modules
   provide adapters (e.g., `internal/persist/userjson`) that satisfy repository
   interfaces. Swap the adapter and register it under the same service key to migrate to
   a database or distributed cache.
2. **Actor + Service Hybrid**: Time-sensitive gameplay continues inside actors (Proto.Actor),
   while request/response workflows use synchronous services. Modules coordinate the two
   by registering actor handles inside the container.
3. **Dependency Inversion via Service Keys**: Modules interact through descriptive
   service keys (`runtime.ServiceUserService`, `runtime.ServiceGatewayPID`). New modules
   can depend on these keys without direct coupling to constructors.
4. **Lifecycle Management**: `App.Run` ensures that modules stop in reverse order, giving
   networking layers a chance to flush requests before actors shut down. Future modules
   can rely on this deterministic teardown.

## Immediate Next Steps

1. **Authentication Hardening**
   * Replace the JSON password storage with salted hashing and eventually an account
     store (SQL/NoSQL).
   * Issue JWT/refresh tokens from a dedicated authentication module.
2. **Gameplay Service Modules**
   * Extract combat, economy, and progression loops into separate modules that expose
     gRPC or WebSocket endpoints as needed.
   * Introduce an event bus (Kafka/NATS) module for cross-service communication when
     scaling out.
3. **Observability Module**
   * Add structured metrics/log forwarding via a monitoring module that hooks into the
     runtime container.
4. **Testing Strategy**
   * Write integration tests per module by instantiating `runtime.App` with only the
     modules under test and using the container to resolve dependencies.

## Long-Term Evolution

* Move configuration assets out of the binary and into a configuration service or CDN,
  with hot-reload support via module restarts.
* Introduce a command bus to let HTTP handlers dispatch gameplay commands to actors
  without being aware of mailbox implementations.
* Support horizontal scaling by making gateway actors stateless and persisting session
  data in a distributed cache (e.g., Redis) provided by another module.

This plan should serve as the foundation for ongoing refactors while preserving the
existing gameplay logic. Every new feature should arrive as a module that cleanly
publishes its dependencies through the runtime container.
