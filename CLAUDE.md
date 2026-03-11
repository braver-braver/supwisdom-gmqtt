# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Gmqtt is an MQTT v3.x and v5 broker implementation in Go with support for clustering, persistence, and a flexible plugin system. The broker provides hooks for customizing behavior (authentication, ACL, etc.) and includes built-in plugins for admin APIs, Prometheus metrics, authentication, and federation.

## Development Commands

### Building
```bash
# Build the broker binary
make build

# Build for Docker (CGO disabled, Linux target)
make binary

# Build Docker image
make build-docker
```

### Running
```bash
# Run with default configuration
make run

# Or run directly
go run ./cmd/gmqttd start -c ./cmd/gmqttd/default_config.yml

# The broker listens on:
# - 1883: TCP MQTT
# - 8883: WebSocket MQTT
# - 8082: Prometheus metrics
# - 8083: HTTP API (gRPC gateway)
# - 8084: gRPC API
```

### Testing
```bash
# Run all tests with race detection
go test -race ./...

# Or use make
make test

# Generate test coverage
make test-cover

# Run benchmarks
make test-bench
```

### Code Generation
```bash
# Generate all code (plugins, mocks, etc.)
go generate ./...

# Or use make
make go-generate

# Generate mocks only
make generate-mocks
# (runs ./mock_gen.sh which uses gomock)
```

### Plugin Development
```bash
# Install the CLI tool
go install ./cmd/gmqctl

# Generate a new plugin template
gmqctl gen plugin -n <plugin_name> -H <hooks> -c <needs_config> -o ./plugin

# Example: Create plugin with OnBasicAuth and OnSubscribe hooks
gmqctl gen plugin -n awesome -H OnBasicAuth,OnSubscribe -c true -o ./plugin
```

### Code Quality
```bash
# Format code
make fmt

# Run linter
make lint

# Run vet
make vet
```

## Architecture

### Core Components

**Server** (`server/server.go`): The main broker implementation. The `Server` interface provides methods for publishing messages, managing clients, and accessing configuration. Plugins interact with the server through this interface.

**Client** (`server/client.go`): Represents an MQTT client connection. Handles packet processing, session management, and message delivery.

**Hooks** (`server/hooks.go`): Extension points for customizing broker behavior. Hooks are called at specific points in the MQTT lifecycle (connect, subscribe, publish, etc.). Plugins implement hooks through wrapper functions.

**Plugin System** (`server/plugin.go`): Plugins implement the `Plugin` interface with `Load()`, `Unload()`, `HookWrapper()`, and `Name()` methods. Plugins are registered via `RegisterPlugin()` and loaded in the order specified in the configuration's `plugin_order` field.

### Persistence Layer

The persistence layer supports both in-memory and Redis backends, controlled by the `persistence.type` configuration:

- **Session** (`persistence/session/`): Stores session state (client ID, clean session flag, expiry, etc.)
- **Subscription** (`persistence/subscription/`): Manages topic subscriptions using a trie structure for efficient wildcard matching
- **Queue** (`persistence/queue/`): Message queues for QoS 1/2 messages and offline message storage
- **Unack** (`persistence/unack/`): Tracks unacknowledged messages (inflight messages)

Each component has memory (`mem/`) and Redis (`redis/`) implementations.

### Packet Handling

**Packets** (`pkg/packets/`): MQTT v3.x and v5 packet encoding/decoding. Each packet type (CONNECT, PUBLISH, SUBSCRIBE, etc.) has its own file with marshal/unmarshal methods.

### Built-in Plugins

- **admin** (`plugin/admin/`): Provides gRPC and HTTP REST APIs for managing the broker (client management, subscription queries, message publishing)
- **auth** (`plugin/auth/`): Username/password authentication with configurable hash types (plain, md5, sha256, bcrypt)
- **prometheus** (`plugin/prometheus/`): Exposes broker metrics for Prometheus scraping
- **federation** (`plugin/federation/`): Clustering support using Serf for gossip protocol and gRPC for inter-node communication

### Configuration

Configuration is loaded from YAML files (default: `$HOME/gmqtt.yml` or via `-c` flag). See `cmd/gmqttd/default_config.yml` for all available options. Key sections:
- `listeners`: TCP and WebSocket endpoints
- `api`: gRPC and HTTP API endpoints
- `mqtt`: MQTT protocol settings (QoS, session expiry, message limits, etc.)
- `persistence`: Storage backend (memory or redis)
- `plugins`: Plugin-specific configuration
- `plugin_order`: Controls plugin loading order

## Testing Requirements

When contributing code:
- Write unit tests for new functionality using testify for assertions
- Run tests with race detection: `go test -race ./...`
- Generate mocks using GoMock (see `mock_gen.sh` for examples)
- Mock files follow the pattern `<source_file>_mock.go`
- Integration tests use [paho.mqtt.testing](https://github.com/eclipse/paho.mqtt.testing)

## Key Patterns

**Hook Wrappers**: Plugins don't implement hooks directly. Instead, they return wrapper functions that wrap the next hook in the chain, allowing multiple plugins to process the same event.

**Persistence Factories**: Persistence implementations are registered via `RegisterPersistenceFactory()` and instantiated based on configuration.

**Topic Matching**: Subscription matching uses a trie data structure (`persistence/subscription/mem/topic_trie.go`) for efficient wildcard topic filtering.

**Message Flow**: Published messages go through `OnMsgArrived` hooks, then are matched against subscriptions, queued for delivery, and finally delivered through `OnDelivered` hooks.
