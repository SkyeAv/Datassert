# Datassert

### By Skye Lane Goetz

Datassert is a high-performance CLI for building a DuckDB-backed assertion store from Babel export files, with a focus on fast local builds and simple command-driven workflows.

## Quick Start

```bash
# Install CLI from GitHub
go install github.com/SkyeAv/datassert@latest

# Verify install
datassert --help
```

## Build Command

```bash
# Build a Datassert database from Babel exports
datassert build --babel-dir /path/to/babel
```

### Flags

| Flag | Required | Default | Description |
| --- | --- | --- | --- |
| `--babel-dir` | Yes | N/A | Directory containing Babel `*Class.ndjson.zst` and `*Synonyms.ndjson.zst` files |
| `--db-path` | No | `./datassert.duckdb` | Output path for the DuckDB database |
| `--batch-size` | No | `1000000` | Number of records written per Parquet batch |

### Input Expectations

- `--babel-dir` is scanned for files matching `*Class.ndjson.zst` and `*Synonyms.ndjson.zst`.
- File matching is non-recursive (top-level of the provided directory).

### Output Artifacts

- Staging Parquet files are written to `./.parquet-store/`.
- Final DuckDB database is written to `--db-path`.
- Build creates and loads `SOURCES`, `CATEGORIES`, `CURIES`, and `SYNONYMS`, then indexes/sorts synonyms for query performance.

### Examples

```bash
# Use defaults for db path and batch size
datassert build --babel-dir ./babel-exports

# Write database to a custom location
datassert build --babel-dir ./babel-exports --db-path ./data/datassert.duckdb

# Tune Parquet batch size
datassert build --babel-dir ./babel-exports --batch-size 500000
```

### Runtime Behavior

- Displays progress bars for class and synonym processing phases.
- Uses CPU-based concurrency (`NumCPU()/2` for class processing and `NumCPU()/4` for synonym processing).

## Maintainer

Skye Lane Goetz
