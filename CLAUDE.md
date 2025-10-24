# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a Chinese-themed idle MMORPG (修仙放置) server with a Vue.js frontend. The project implements a cultivation/xianxia themed idle game where players practice different "sequences" (修炼序列) to gain resources and experience.

## Architecture

The project follows a **pure Actor-driven + DDD (Domain-Driven Design)** architecture:

- **Backend**: Go with protoactor-go Actor system
- **Frontend**: Vue 3 + Vite + Pinia + Vue Router
- **Communication**: WebSocket for real-time, HTTP for authentication
- **Persistence**: JSON file-based storage (with plans for Redis/PostgreSQL)

### Key Architectural Patterns

1. **Actor Model**: All game logic is encapsulated in independent Actors (GatewayActor, PlayerActor, SequenceActor, PersistActor)
2. **Domain-Driven Design**: Business logic is abstracted into domain objects (Sequence, Formula, Item, Inventory)
3. **Message-Driven**: Components communicate only through message passing, no shared memory
4. **Async Persistence**: Persistence handled through PersistActor to avoid blocking game logic
5. **Table-Driven Configuration**: All game data driven by JSON configuration tables

## Common Development Commands

### Frontend (idle-vue/)
```bash
cd idle-vue
npm install          # Install dependencies
npm run dev         # Start development server (port 5173)
npm run build       # Build for production
npm run preview     # Preview production build
```

### Backend (idlemmoserver/)
```bash
cd idlemmoserver
go mod tidy         # Download dependencies
go run cmd/server/main.go    # Start server (port 8080)
```

## Project Structure

```
idle-server/
├── idle-vue/                 # Vue.js frontend
│   ├── src/
│   │   ├── api/             # HTTP and WebSocket clients
│   │   ├── store/           # Pinia state management
│   │   ├── views/           # Vue components/pages
│   │   └── router/          # Vue Router configuration
│   └── package.json
├── idlemmoserver/            # Go backend
│   ├── cmd/server/          # Application entry point
│   ├── internal/
│   │   ├── actors/          # Actor layer (core game logic)
│   │   ├── domain/          # Domain layer (DDD entities)
│   │   ├── gateway/         # HTTP + WebSocket layer
│   │   ├── persist/         # Persistence layer
│   │   └── config/          # Configuration
│   ├── saves/               # JSON player save files
│   └── go.mod
└── README.md
```

## Core Game Concepts

### Sequences (修炼序列)
- Players choose different sequences for idle cultivation
- Examples: herb gathering, mining, alchemy, meditation
- Each sequence has independent levels, experience, and rewards
- Sequences produce resources, items, and rare events on ticks

### Actor System
- **GatewayActor**: Handles WebSocket/HTTP connections and routing
- **PlayerActor**: Manages player state and coordinates other actors
- **SequenceActor**: Handles sequence logic and tick calculations
- **PersistActor**: Asynchronous save/load operations
- **SchedulerActor**: (Planned) Unified tick scheduling

### Message Flow
1. Client connects via WebSocket with auth token
2. GatewayActor routes messages to appropriate PlayerActor
3. PlayerActor spawns/manages SequenceActor for active sequences
4. SequenceActor sends tick results to PlayerActor
5. PlayerActor forwards to PersistActor for async saving
6. Results broadcasted back to client via WebSocket

## Key Files and Their Roles

### Backend Core Files
- `cmd/server/main.go`: Server bootstrap and ActorSystem initialization
- `internal/actors/messages.go`: All Actor message definitions
- `internal/actors/player_actor.go`: Player state management
- `internal/actors/sequence_actor.go`: Sequence tick logic
- `internal/actors/persist_actor.go`: Async save/load operations
- `internal/domain/sequence.go`: Sequence domain logic
- `internal/domain/items.go`: Item and inventory systems
- `internal/gateway/ws.go`: WebSocket connection handling
- `internal/gateway/http.go`: HTTP authentication endpoints

### Frontend Core Files
- `src/main.js`: Vue app initialization
- `src/api/ws.js`: WebSocket client with auto-reconnect
- `src/api/http.js`: HTTP client for authentication
- `src/store/user.js`: Player state management (Pinia)
- `src/views/`: Game UI components

### Configuration
- `internal/domain/config.json`: Game configuration tables (sequences, items, drops)
- `saves/`: Player JSON save files

## Development Guidelines

### Backend Development
- All Actor communication must use protoactor-go message passing
- Actors should be single-threaded and stateless where possible
- Use the message definitions in `internal/actors/messages.go`
- Game logic should be in domain layer, not Actor layer
- All persistence operations must go through PersistActor

### Frontend Development
- Use Pinia for state management
- WebSocket communication handled by `src/api/ws.js`
- Follow Vue 3 Composition API patterns
- All game UI should be reactive to store state changes

### Testing the Full System
1. Start backend: `cd idlemmoserver && go run cmd/server/main.go`
2. Start frontend: `cd idle-vue && npm run dev`
3. Open browser to `http://localhost:5173`
4. Login and test sequence functionality

## Current Implementation Status

The project is in Phase 1 development with core idle loop implemented:
- ✅ HTTP authentication
- ✅ WebSocket real-time communication
- ✅ Basic Actor system
- ✅ Sequence tick logic and rewards
- ✅ Inventory system
- ✅ JSON file persistence
- ✅ CORS support for frontend

## Next Development Priorities

- SchedulerActor for unified tick management
- Sequence leveling and progression systems
- Bag management UI and commands
- Multiplayer features (future phases)
- Database persistence (Redis/PostgreSQL)