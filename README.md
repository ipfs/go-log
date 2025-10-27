# go-log

[![](https://img.shields.io/badge/made%20by-Protocol%20Labs-blue.svg?style=flat-square)](https://protocol.ai)
[![](https://img.shields.io/badge/project-IPFS-blue.svg?style=flat-square)](https://ipfs.io/)
[![GoDoc](https://pkg.go.dev/badge/github.com/ipfs/go-log/v2.svg)](https://pkg.go.dev/github.com/ipfs/go-log/v2)

> The logging library used by go-ipfs

go-log wraps [zap](https://github.com/uber-go/zap) to provide a logging facade. go-log manages logging
instances and allows for their levels to be controlled individually.

## Install

```sh
go get github.com/ipfs/go-log
```

## Usage

Once the package is imported under the name `logging`, an instance of `EventLogger` can be created like so:

```go
var log = logging.Logger("subsystem name")
```

It can then be used to emit log messages in plain printf-style messages at seven standard levels:

Levels may be set for all loggers:

```go
lvl, err := logging.LevelFromString("error")
if err != nil {
	panic(err)
}
logging.SetAllLoggers(lvl)
```

or individually:

```go
err := logging.SetLogLevel("net:pubsub", "info")
if err != nil {
	panic(err)
}
```

or by regular expression:

```go
err := logging.SetLogLevelRegex("net:.*", "info")
if err != nil {
	panic(err)
}
```

### Environment Variables

This package can be configured through various environment variables.

#### `GOLOG_LOG_LEVEL`

Specifies the log-level, both globally and on a per-subsystem basis.

For example, the following will set the global minimum log level to `error`, but reduce the minimum
log level for `subsystem1` to `info` and reduce the minimum log level for `subsystem2` to debug.

```bash
export GOLOG_LOG_LEVEL="error,subsystem1=info,subsystem2=debug"
```

`IPFS_LOGGING` is a deprecated alias for this environment variable.

#### `GOLOG_FILE`

Specifies that logs should be written to the specified file. If this option is _not_ specified, logs are written to standard error.

```bash
export GOLOG_FILE="/path/to/my/file.log"
```

#### `GOLOG_OUTPUT`

Specifies where logging output should be written. Can take one or more of the following values, combined with `+`:

- `stdout` -- write logs to standard out.
- `stderr` -- write logs to standard error.
- `file` -- write logs to the file specified by `GOLOG_FILE`

For example, if you want to log to both a file and standard error:

```bash
export GOLOG_FILE="/path/to/my/file.log"
export GOLOG_OUTPUT="stderr+file"
```

Setting _only_ `GOLOG_FILE` will prevent logs from being written to standard error.

#### `GOLOG_LOG_FMT`

Specifies the log message format. It supports the following values:

- `color` -- human readable, colorized (ANSI) output
- `nocolor` -- human readable, plain-text output.
- `json` -- structured JSON.

For example, to log structured JSON (for easier parsing):

```bash
export GOLOG_LOG_FMT="json"
```

The logging format defaults to `color` when the output is a terminal, and `nocolor` otherwise.

`IPFS_LOGGING_FMT` is a deprecated alias for this environment variable.

#### `GOLOG_LOG_LABELS`

Specifies a set of labels that should be added to all log messages as comma-separated key-value
pairs. For example, the following add `{"app": "example_app", "dc": "sjc-1"}` to every log entry.

```bash
export GOLOG_LOG_LABELS="app=example_app,dc=sjc-1"
```

#### `GOLOG_CAPTURE_DEFAULT_SLOG`

When `SetupLogging()` is called, go-log automatically routes slog logs through its zap core for consistent formatting and dynamic level control (unless explicitly disabled). This means libraries using `slog` (like go-libp2p) will automatically use go-log's formatting and respect dynamic level changes (e.g., via `ipfs log level` commands).

To disable this behavior and keep `slog.Default()` unchanged, set:

```bash
export GOLOG_CAPTURE_DEFAULT_SLOG="false"
```

### Slog Integration

go-log automatically integrates with Go's `log/slog` package when `SetupLogging()` is called. This provides:

1. **Unified formatting**: slog logs use the same format as go-log (color/nocolor/json)
2. **Dynamic level control**: slog loggers respect `SetLogLevel()` and environment variables
3. **Subsystem-aware filtering**: slog loggers with subsystem attributes get per-subsystem level control

**Note**: This slog bridge exists as an intermediate solution while go-log uses zap internally. In the future, go-log may migrate from zap to native slog, which would simplify this integration.

#### How it works

Libraries like go-libp2p use gologshim to create slog loggers. When these loggers detect go-log's slog bridge, they automatically integrate with go-log's level control.

**Attributes added by gologshim:**
- `logger`: Subsystem name (e.g., "ping", "swarm2", "basichost")
- Any additional labels from `GOLOG_LOG_LABELS`

Example from go-libp2p's ping protocol:
```go
var log = logging.Logger("ping")  // gologshim
log.Debug("ping error", "err", err)
```

Output when formatted by go-log (JSON format shown here, also supports color/nocolor):
```json
{
  "level": "debug",
  "ts": "2025-10-27T12:34:56.789+0100",
  "logger": "ping",
  "caller": "ping/ping.go:72",
  "msg": "ping error",
  "err": "connection refused"
}
```

#### Controlling slog logger levels

These loggers respect go-log's level configuration:

```bash
# Via environment variable (before daemon starts)
export GOLOG_LOG_LEVEL="error,ping=debug"

# Via API (while daemon is running)
logging.SetLogLevel("ping", "debug")
```

This works even if the logger is created lazily or hasn't been created yet. Level settings are preserved and applied when the logger is first used.

#### For library authors

If you're writing a library that uses `log/slog` and want to integrate with go-log's level control system, you can detect go-log's slog bridge using duck typing to avoid including go-log in your library's go.mod:

```go
// Check if slog.Default() is go-log's bridge
type goLogBridge interface {
    GoLogBridge()
}

if _, ok := slog.Default().Handler().(goLogBridge); ok {
    // go-log's bridge is active - use it for consistent formatting
    // and dynamic level control via WithAttrs to add subsystem name
    h := slog.Default().Handler().WithAttrs([]slog.Attr{
        slog.String("logger", "mysubsystem"),
    })
    return slog.New(h)
}

// Fallback: create your own slog handler
```

This pattern allows libraries to integrate without adding go-log as a dependency. The `GoLogBridge()` marker method is implemented by both `zapToSlogBridge` and `subsystemAwareHandler` types in go-log's slog bridge.

For a complete example, see [go-libp2p's gologshim](https://github.com/libp2p/go-libp2p/blob/master/gologshim/gologshim.go).

#### Disabling slog integration

To disable automatic slog integration and keep `slog.Default()` unchanged:

```bash
export GOLOG_CAPTURE_DEFAULT_SLOG="false"
```

When disabled, go-libp2p's gologshim will create its own slog handlers that write to stderr.

## Contribute

Feel free to join in. All welcome. Open an [issue](https://github.com/ipfs/go-log/issues)!

This repository falls under the IPFS [Code of Conduct](https://github.com/ipfs/community/blob/master/code-of-conduct.md).

### Want to hack on IPFS?

[![](https://cdn.rawgit.com/jbenet/contribute-ipfs-gif/master/img/contribute.gif)](https://github.com/ipfs/community/blob/master/CONTRIBUTING.md)

## License

MIT
