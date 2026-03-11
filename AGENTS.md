# Repository Guidelines

## Project Structure & Module Organization
Core broker logic lives in `server/`, with protocol and shared utilities in `pkg/` (for example `pkg/packets`). Entry points are under `cmd/`: `cmd/gmqttd` (broker) and `cmd/gmqctl` (CLI tooling). Built-in plugins are in `plugin/` (`admin`, `auth`, `federation`, `prometheus`). Stateful backends are grouped in `persistence/`, retained-message logic in `retained/`, and topic-alias logic in `topicalias/`. Configuration and fixtures are in `config/` and `*/testdata`.

## Build, Test, and Development Commands
- `make run`: generate code, then run broker with `cmd/gmqttd/default_config.yml`.
- `make build`: run `go generate` and build `build/gmqttd`.
- `make test`: regenerate mocks and run `go test -v -race ./...`.
- `make test-cover` or `./go_test.sh`: produce coverage output (filters generated/mock/example files).
- `make generate-mocks`: refresh GoMock files using `mock_gen.sh`.
- `make fmt` and `make vet`: run `goimports` formatting and `go vet` checks.

For quick local verification, use `go test -race ./...` (same baseline as CI).

## Coding Style & Naming Conventions
This is a Go codebase; follow Go conventions and [CodeReviewComments](https://github.com/golang/go/wiki/CodeReviewComments). Format with `goimports` (tabs for indentation, grouped imports). Keep package names lowercase and short. Use descriptive file names by feature area (for example `server/client.go`, `plugin/auth/hooks.go`). Generated mocks must be named `*_mock.go` and stay in the owning package.

## Testing Guidelines
Place tests beside implementation files using `*_test.go` (examples: `server/server_test.go`, `plugin/auth/auth_test.go`). Prefer table-driven tests and `testify` assertions. Any behavior change should include unit tests. Before opening a PR, run `make test`; for protocol-sensitive changes, also run race-enabled build/tests and relevant `paho.mqtt.testing` interoperability cases.

## Commit & Pull Request Guidelines
Commits in history are short and imperative, often with optional prefixes (`fix: ...`, `docs: ...`, `fix(federation): ...`). Keep subject lines focused on one change. Pull requests should:
- Link to at least one open issue.
- Explain behavior changes and impacted modules/plugins.
- Include test evidence (commands run, notable output).
- Discuss design first in an issue for larger features.
