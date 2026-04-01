# Agent Development Guide

Guidelines for agentic coding agents working on the Datassert codebase — a high-performance Go CLI that processes Babel export files into sharded DuckDB databases.

## Build & Run

```bash
go build -o datassert                              # Compile binary
go run main.go                                     # Run without compiling
go run main.go build --babel-dir <dir>             # Build pipeline (default --db-dir: ".")
```

The `build` command flags:

| Flag | Default | Description |
|------|---------|-------------|
| `--babel-dir` | (required) | Directory containing Babel export files |
| `--db-dir` | `.` | Base output path for sharded DuckDB databases |
| `--batch-size` | `100000` | Records per Parquet batch flush |
| `--buffer-size` | `2048` | Channel buffer size for synonym record streaming |
| `--class-cpu-fraction` | `2` | CPU count divided by this = class ingest goroutines |
| `--synonym-cpu-fraction` | `4` | CPU count divided by this = synonym worker goroutines |

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
│   └── build.go      # build subcommand + all pipeline logic (704 lines)
```

All application logic lives in `cmd/build.go`. The `cmd` package uses `package cmd`.

## Pipeline Architecture

The `build` command runs three sequential phases:

1. **Class ingest** — Reads `*Class.ndjson.zst` files in parallel via `errgroup`. Builds a `ClassLookup` (sharded map of curie hash → tab-joined aliases). Memory-only, no disk output.
2. **Synonym processing** — Reads `*Synonyms.ndjson.zst` files sequentially. For each file, a producer goroutine decodes records into a buffered channel; N worker goroutines consume records, assign IDs via sharded atomic counters, and batch-flush to intermediate Parquet files via `writeIfGtLen`. Also generates L1 (normalized) synonyms by stripping non-word characters, and writes categories/sources tables.
3. **DuckDB assembly** — For each of 16 shards, reads the shard's Parquet globs into DuckDB tables (with ORDER BY), creates indexes, and runs `VACUUM ANALYZE`. Runs sequentially across shards.

Output directory structure: `<db-dir>/datassert/.parquets/` (staging) and `<db-dir>/datassert/data/{0..15}.duckdb`. The output directory is fully removed and recreated on each run (`mkDirs` calls `os.RemoveAll`).

## Data Schemas

### Input Records

```go
type ClassRecord struct {
    EquivalentIdentifiers []string `json:"equivalent_identifiers"`
}

type SynonymRecord struct {
    Curie         string   `json:"curie"`
    Synonyms      []string `json:"names"`
    PreferredName string   `json:"preferred_name"`
    Categories    []string `json:"types"`
    Taxon         []any    `json:"taxa"`
}
```

### Parquet Tables

```go
type CuriesTable struct {
    CurieID       uint32 `parquet:"CURIE_ID"`
    Curie         string `parquet:"CURIE"`
    PreferredName string `parquet:"PREFERRED_NAME"`
    CategoryID    uint32 `parquet:"CATEGORY_ID"`
    Taxon         uint32 `parquet:"TAXON_ID,optional"`
}

type SynonymsTable struct {
    CurieID  uint32 `parquet:"CURIE_ID"`
    SourceID uint8  `parquet:"SOURCE_ID"`
    Synonym  string `parquet:"SYNONYM"`
}

type CategoriesTable struct {
    CategoryID uint32 `parquet:"CATEGORY_ID"`
    Category   string `parquet:"CATEGORY_NAME"`
}

type SourcesTable struct {
    SourceID      uint8  `parquet:"SOURCE_ID"`
    SourceName    string `parquet:"SOURCE_NAME"`
    SourceVersion string `parquet:"SOURCE_VERSION"`
    NLPLevel      uint8  `parquet:"NLP_LEVEL"`
}
```

`SourcesTable` is hardcoded as two rows: SourceID 0 (NLPLevel 0) and SourceID 1 (NLPLevel 1), both `"BABEL"` / `"SEPT-2025"`.

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

No `error` return values — use `checkError(code, err)` / `throwError(code, err)` with `uint8` site codes, which call `log.Fatalf`. Do not introduce `error` return patterns. Each call site gets a unique code; when adding new call sites, use the next available code. Currently codes 1–15 are used; next available is 16.

### Struct Tags

- **JSON**: `snake_case` field names (`json:"equivalent_identifiers"`, `json:"names"`, `json:"types"`, `json:"taxa"`)
- **Parquet**: `ALL_CAPS` with optional tags (`parquet:"TAXON_ID,optional"`)

### Concurrency

- **Sharded structures** — `[nShards]` (16) with per-shard mutexes and `_pad [40]byte` to avoid false sharing. Initialize inner maps in a loop before use. Used by `ClassLookup` and `CurieCounter`.
- **Sharded sync.Map** — `CategoryMap` uses `[nShards]` sharding with `sync.Map` per shard for lock-free reads, plus `atomic.Uint32` counter for ID assignment via `LoadOrStore`.
- **Double-checked locking** — for read-heavy patterns (`CurieCounter.GetOrNext`), `RLock` fast-path then `Lock` slow-path with re-check.
- **Bounded parallelism** — `errgroup.Group` with `g.SetLimit(n)` over raw `sync.WaitGroup`.
- **Producer-consumer** — buffered channels (sized by `bufferSize`); producer decodes into channel and closes on EOF.
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
- Configure: `SET temp_directory`, `SET preserve_insertion_order = false`
- Load (with ordering):
  - `SOURCES` — no ordering
  - `CATEGORIES` — `ORDER BY CATEGORY_NAME`
  - `CURIES` — `ORDER BY TAXON_ID`
  - `SYNONYMS` — `ORDER BY SYNONYM`
- Indexes: `CURIE_SYNONYMS ON SYNONYMS(SYNONYM)`, `CATEGORY_NAMES ON CATEGORIES(CATEGORY_NAME)`, `CURIE_TAXON ON CURIES(TAXON_ID)`
- Finalize: `VACUUM ANALYZE`

## Key Behaviors

### Token Cleaning Pipeline

`cleanAliases` lowercases, trims, and deduplicates aliases. `cleanToken` iteratively strips wrapping quotes/double-quotes and leading doubled prefixes. `isBadToken` rejects empty strings, tokens containing `"INCHIKEY"` or `"inchikey"`, and exact matches for `"uncharacterized protein"`, `"uncharacterized gene"`, or `"hypothetical protein"`.

### L0/L1 Synonym Generation

For each synonym record, the pipeline produces two levels:
- **L0 synonyms** (SourceID 0) — the original cleaned, deduplicated synonyms plus class aliases
- **L1 synonyms** (SourceID 1) — generated by stripping non-word characters (`\W+`) from L0 synonyms, excluding any that already exist in the L0 set

### ClassLookup Draining

`ClassLookup.Get` is destructive — it deletes the entry after reading. This is intentional: each curie's class aliases are consumed once during synonym processing and then removed from memory.

### Taxon Processing

Taxon values from `SynonymRecord.Taxon` (`[]any`) are processed by taking the first element, converting to string, stripping the `"NCBITaxon:"` prefix, and parsing as integer.

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
