# Benchmark Plugin

This plugin provides performance benchmarking capabilities for GoREST APIs.

## Structure

```
plugins/benchmark/
├── benchmark.go           # Plugin implementation
├── cmd/
│   └── main.go           # CLI entry point
├── testserver/
│   └── main.go           # Test server for benchmarks
└── README.md             # This file
```

## CLI Installation

The benchmark plugin provides its own command-line interface via `/cmd/benchmark` (symlinked from `plugins/benchmark/cmd`).

### Usage

```bash
# Run via make
make benchmark

# Run directly
go run ./cmd/benchmark/main.go

# Run from plugin directory
go run ./plugins/benchmark/cmd/main.go
```

## How It Works

1. The plugin implements the `CommandProvider` interface
2. It registers itself via `init()` function for auto-discovery
3. The CLI in `cmd/main.go` discovers and executes the plugin
4. A symlink from `/cmd/benchmark` makes it accessible from the main commands directory
5. During execution, the plugin builds and runs the testserver to perform benchmarks

## Features

- Generates benchmark test data
- Builds and runs a dedicated test server
- Measures API endpoint performance
- Tests various concurrency levels (1, 10, 50)
- Tests different data sizes (10, 100, 1000 records)
- Reports response times (p50, p95, p99)
- Tracks error rates
- Automatically cleans up and restores original schema
