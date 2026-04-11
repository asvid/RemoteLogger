# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Android library + server(s) for streaming Logcat over WebSocket to a developer machine — useful when device must run in release mode without developer options.

**Flow:** Android app reads logcat → sends JSON events via WebSocket → server displays/filters/saves logs.

## Components

| Component | Language | Purpose |
|---|---|---|
| `RemoteLoggerLib/` | Kotlin | Android library |
| `ExampleApp/` | Kotlin | Demo Android app |
| `serverGo/` | Go | TUI server — bubbletea terminal UI, WebSocket, log filtering |

## Commands

### Android (from repo root)
```bash
./gradlew build                                    # build all modules
./gradlew :RemoteLoggerLib:build                   # build library only
./gradlew :ExampleApp:installDebug                 # deploy example app to device
./gradlew test                                     # run unit tests
./gradlew :RemoteLoggerLib:publishToMavenLocal     # publish library locally
```

### Go server
```bash
cd serverGo
go build -o remotelogger .                         # build binary
go run .                                           # run without building
./remotelogger -p 3000 -ll ERROR -t MyTag -f output.log
./remotelogger -no-tui                             # plain stdout, suitable for piping/CI
go mod tidy                                        # sync dependencies
```

## Architecture

### Go Server (`serverGo/`)

Three packages:

- **`logentry/`** — `Entry` struct + `Level` type. `ParseLevel` returns `(Level, bool)`. `Level.UnmarshalJSON` handles Android's string-encoded levels.
- **`ws/`** — WebSocket server. Emits typed `Event{Kind, Entry, Addr}` on a single `chan Event`. Kinds: `EventLog`, `EventConnected`, `EventDisconnected`.
- **`tui/`** — Bubbletea model/update/view split across four files:
  - `model.go` — `Model` struct, `New()`, `appendLog()`, `applyFilter()`, `Init()`
  - `update.go` — `Update()`: handles `serverEvent`, `tea.KeyMsg`, `tea.WindowSizeMsg`
  - `view.go` — `View()`: header bar, table, filter bar, status line; level colors via lipgloss
  - `filter.go` — `Filter` struct with `matches()`, `nextLevel()` (cycles ALL→VERBOSE→DEBUG→INFO→WARNING→ERROR→ALL)

**Data flow:** WS goroutine → `chan ws.Event` → `waitForEvent` tea.Cmd → `serverEvent` tea.Msg → `model.Update()` → `appendLog()`

**Buffer cap:** `logs` capped at 50k entries; trimmed to 40k when exceeded, then `applyFilter()` rebuilds `display`. Two parallel slices (`display []Entry`, `rows []table.Row`) avoid O(n) row rebuild on each new log — only rebuilt on filter change or trim.

**TUI keybindings:** `/` search · `t` tag filter · `l` cycle level · `c` clear filters · `s` save snapshot to timestamped file · `G` jump to latest (re-enables auto-scroll) · `q`/`ctrl+c` quit · `↑`/`pgup` disables auto-scroll

**`-no-tui` flag:** skips bubbletea, prints colored logs to stdout. Useful for `|grep` pipelines or CI.

### Android Library (`RemoteLoggerLib/`)

Single file: `RemoteLogger.kt`. Key design:

- `initialize(Config)` — starts logcat reader coroutine and `connectWithBackoff()` coroutine on a shared `CoroutineScope(SupervisorJob() + Dispatchers.IO)`.
- **Logcat reader** — runs `logcat <packageName> -v time`, parses each line with regex, enqueues `Event` into an unlimited `Channel` via `trySend`.
- **WebSocket** — OkHttp `WebSocketListener`. On `onOpen`, drains the channel and forwards via `drainQueueToWs()`. On `onFailure`/`onClosed`, reconnects via `connectWithBackoff()`.
- **Reconnect** — exponential backoff from `reconnectDelayMs` (default 3s), capped at 30s. Reset to initial delay on successful connect.
- `Config.sendHistoryOnConnect = false` clears the event buffer instead of replaying it on reconnect.
- Crash handler sends crash event then `exitProcess(1)`.

### Library Publishing

`RemoteLoggerLib/build.gradle.kts` uses `com.vanniktech.maven.publish` v0.30.0. Version/coordinates are set via `coordinates("io.github.asvid", "remote-logger", "1.0.0")` inside `mavenPublishing {}`. Publishes to Maven Central via Sonatype Central Portal. Requires `ORG_GRADLE_PROJECT_mavenCentralUsername`, `ORG_GRADLE_PROJECT_mavenCentralPassword`, and GPG signing properties in `~/.gradle/gradle.properties` or env vars.

## Event JSON Schema

```json
{ "tag": "MyTag", "message": "log message", "level": "INFO", "time": "04-11 12:34:56.789" }
```
