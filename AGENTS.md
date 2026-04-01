# Agent Development Guide

Guidelines for agentic coding agents working on the Datassert codebase — a high-performance Go CLI that processes Babel export files into sharded DuckDB databases.

## Build & Run

```bash
go build -o datassert                              # Compile binary
go run main.go                                     # Run without compiling
go run main.go build --babel-dir <dir>             # Build pipeline (default --db-dir: ".")
```

The `build` command also accepts: `--db-dir`, `--batch-size` (default 100000), `--buffer-size` (default 2048), `--class-cpu-fraction` (default 2), `--synonym-cpu-fraction` (default 4).

## Testing

```bash
go test ./...                      # All tests
go test -v ./...                   # Verbose
go test -v ./cmd/                  # Specific package
go test -run TestFoo ./cmd/        # Single test by name
go test -cover ./...               # With coverage
```

No test files exist yet. When adding tests, place `*_test.go` files alongside the code they test (same package).

## Linting & Formatting

```bash
go fmt ./...                       # Format all
go vet ./...                       # Static analysis
```

## Project Structure

```
Datassert/
├── main.go           # Entry point → cmd.Execute()
├── go.mod            # Module: github.com/SkyeAv/datassert (Go 1.25.7)
├── cmd/
│   ├── root.go       # Root cobra command ("datassert")
│   └── build.go      # build subcommand + all pipeline logic (~700 lines)
```

All application logic lives in `cmd/build.go`. The `cmd` package uses `package cmd`.

## Pipeline Architecture

The `build` command runs three sequential phases:

1. **Class ingest** — Reads `*Class.ndjson.zst` files in parallel via `errgroup`. Builds a `ClassLookup` (sharded map of curie → aliases). Memory-only, no disk output.
2. **Synonym processing** — Reads `*Synonyms.ndjson.zst` files sequentially. For each file, a producer goroutine decodes records into a buffered channel; N worker goroutines consume records, assign IDs via sharded atomic counters, and batch-flush to intermediate Parquet files via `writeIfGtLen`.
3. **DuckDB assembly** — For each of 16 shards, reads the shard's Parquet glob into DuckDB tables, creates indexes, and runs `VACUUM ANALYZE`.

Output directory structure: `<db-dir>/datassert/.parquets/` (staging) and `<db-dir>/datassert/data/{0..15}.duckdb`.

## Code Style

### Imports

Standard library (alphabetical), blank line, third-party (alphabetical). Each import on its own line.

### Naming

| Element | Convention | Example |
|---------|-----------|---------|
| Types/Structs | PascalCase | `ClassLookup`, `CurieCounter` |
| Interfaces | PascalCase | `ParquetTable` |
| Functions | PascalCase (exported), camelCase (unexported) | `WriteParquet`, `checkError` |
| Variables | camelCase | `batchSize`, `babelDir` |
| Constants | camelCase or PascalCase | `nShards`, `badPrefixes` |

### Error Handling

No `error` return values — use `checkError(code, err)` / `throwError(code, err)` with `uint8` site codes (0–15 range), which call `log.Fatalf`. Do not introduce `error` return patterns. Each call site gets a unique code; when adding new call sites, use the next available code.

### Struct Tags

- **JSON**: `snake_case` field names (`json:"equivalent_identifiers"`)
- **Parquet**: `ALL_CAPS` with optional tags (`parquet:"TAXON_ID,optional"`)

### Concurrency

- **Sharded structures** — `[nShards]` (16) with per-shard mutexes and `_pad [40]byte` to avoid false sharing. Initialize inner maps in a loop before use.
- **Double-checked locking** — for read-heavy patterns (`CurieCounter.GetOrNext`), `RLock` fast-path then `Lock` slow-path with re-check.
- **Bounded parallelism** — `errgroup.Group` with `g.SetLimit(n)` over raw `sync.WaitGroup`.
- **Producer-consumer** — buffered channels; producer decodes into channel and closes on EOF.
- **Lock-free maps** — `sync.Map` for read-heavy concurrent maps (e.g., `CategoryMap`).
- **Shared counters** — `atomic.Uint32`.

### Resource Management

Always `defer Close()` immediately after acquisition (`yieldReader` → `yieldDecoder`).

### JSON Decoding

Use `sonic.ConfigDefault.NewDecoder` for streaming with `io.EOF` break.

### Generics

Constrain with the `ParquetTable` interface union (`CuriesTable | SynonymsTable | CategoriesTable | SourcesTable`). Generic functions like `writeParquet[T]` and `writeIfGtLen[T]` operate across all table types.

### Cobra CLI

Package-level command var, `init()` for registration and flags, `MarkFlagRequired()`.

### Hashing & Sharding

All sharding uses `xxhash.Sum64` modulo `nShards` (16). Use `hashAndShard` helper which returns both the hash and shard number.

### Parquet File Naming

Intermediate Parquet files: `<stem>-<thing>:<fileNum>-<shardNum>:<workerID>.parquet` in the `.parquets/` staging directory.

### DuckDB

- Open: `sql.Open("duckdb", path)` with blank import `_ "github.com/duckdb/duckdb-go/v2"`
- Load: `CREATE TABLE AS SELECT * FROM read_parquet('glob')`
- Configure: `SET temp_directory`, `SET preserve_insertion_order = false`
- After loading: `CREATE INDEX`, `VACUUM ANALYZE`

## Key Dependencies

| Dependency | Purpose |
|-----------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/duckdb/duckdb-go/v2` | DuckDB driver (blank import) |
| `github.com/parquet-go/parquet-go` | Parquet file I/O |
| `github.com/bytedance/sonic` | Fast JSON streaming parser |
| `github.com/klauspost/compress/zstd` | Zstandard decompression |
| `github.com/cespare/xxhash/v2` | XXHash for sharding |
| `github.com/gosuri/uiprogress` | Progress bars |
| `golang.org/x/sync` | errgroup for bounded concurrency |
