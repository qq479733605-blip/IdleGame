# IdleMMO Modular Backend

This directory contains a modular, actor-driven backend split into independent services.  Each service can be started individually or together via the helper script in `scripts/run_all.sh` (requires a running NATS server).

## Services

| Service  | Description |
|----------|-------------|
| `common` | Shared message contracts, constants and helper utilities used across all services. |
| `login`  | REST service for player registration, login and token verification backed by a JSON user repository. |
| `gate`   | WebSocket gateway that authenticates users via the login service and bridges client messages to the game service through NATS. |
| `game`   | Actor-based game logic service.  Player actors are spawned on demand, load/save their state via the persist service and communicate with the gateway through NATS. |
| `persist`| Lightweight persistence service that stores player snapshots to disk in JSON format. |

## Requirements

* Go 1.22+
* Running NATS server (defaults to `nats://127.0.0.1:4222`)

Environment variables allow overriding default addresses:

* `NATS_URL` – NATS connection string used by gate/game/persist services.
* `LOGIN_URL` – HTTP address of the login service (used by the gateway).
* `GATE_ADDR`, `LOGIN_ADDR` – Listen addresses for the gateway and login services.

## Development helpers

```bash
# Run all services (expects NATS to be running locally)
./scripts/run_all.sh
```

Each service exposes a standalone `main.go` so they can be launched individually as needed during development or deployment.
