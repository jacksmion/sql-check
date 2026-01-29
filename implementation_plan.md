# SQL Slow Query Inspection Tool Implementation Plan

## 1. Project Initialization & Architecture Design
- [x] **Initialize Project**
    - `go mod init sql-check`
    - Setup dependencies (cobra, tidb/parser, etc.)
- [x] **Directory Structure Definition**
    - `cmd/sql-check`: Application entry point.
    - `internal/model`: Core data structures (SQLSegment, Risk, Schema definitions).
    - `internal/scanner`: File system traversal and filtering.
    - `internal/extractor`: Polymorphic extractors for different languages.
    - `internal/parser`: Integration with TiDB parser and DDL loading.
    - `internal/auditor`: Rule engine and individual rule implementations.
    - `internal/reporter`: Output formatters (Console, HTML/Markdown).
- [x] **Define Core Interfaces**
    - `Extractor`: Interface for file -> []SQLSegment.
    - `Rule`: Interface for AST check -> []Issue.

## 2. SQL Extraction Module (The Miner)
- [x] **File System Scanner (Concurrent)**
    - Implement a producer-consumer model:
        - **Producer**: Walks the directory tree and sends file paths to a channel.
        - **Worker Pool**: Multiple goroutines reading paths and performing extraction.
    - Implement recursive directory walker.
    - Add support for `.gitignore` or custom exclude patterns.
    - Implement file extension filtering.
- [x] **Generic Extractor**
    - Implement regex-based extraction for generic text files.
    - Pattern matching for `SELECT|UPDATE|DELETE|INSERT` keywords.
- [x] **Language-Specific Extractors**
    - **Go Extractor**: Optimize for Go raw strings (backticks) and double quotes.
    - **Python/C++ Extractor**: Implement logic to handle multi-line strings and variable concatenation awareness (basic level).
- [x] **Location Tracking**
    - Capture absolute file path and line number for every match.


## 3. Environment Context & Parsing (Schema Awareness)
- [x] **DDL Loader**
    - Implement `schema.sql` file reader.
    - Parse `CREATE TABLE` statements to extract:
        - Table Names
        - Column Names & Types
        - Primary Keys & Indexes (including composite indexes)
- [x] **SQL Parsing Wrapper**
    - Integrate `github.com/pingcap/tidb/parser`.
    - Implement error handling for unparseable dynamic SQL (e.g., complex string concatenation).

## 4. Core Audit Engine (The Auditor)
- [x] **Rule Engine Core**
    - Mechanism to register and run rules against parsed AST.
    - Context injection (Schema info).
- [x] **Implement "Fatal" Rules**
    - **Unsafe Write**: Detect `UPDATE` or `DELETE` statements without a `WHERE` clause.
- [x] **Implement "Warning" Rules (Performance)**
    - **Index Miss**: Check `WHERE` columns against the loaded Schema indexes.
    - **Leftmost Prefix**: Verify composite index usage (e.g., if Index(a,b), `WHERE b=1` is a violation).
    - **Implicit Conversion**: Detect type mismatches (e.g., string column compared to int literal).
    - **Deep Pagination**: Detect `LIMIT offset, count` where `offset` is essentially large (e.g., > 5000).
    - **Negative Query**: specific checks for `!=`, `NOT IN`, and `LIKE` starting with wildcards.
- [x] **Implement "Suggestion" Rules**
    - **Select Star**: Flag `SELECT *` usages.

## 5. Reporting Module (The Reporter)
- [x] **Console Output**
    - Implement compiler-style output: `path/to/file.go:12: [WARN] Index missing on column 'user_id'`.
    - Colorized output for different risk levels.
- [x] **Interactive Reports**
    - **Markdown**: Generate a structured markdown table of issues.
    - **HTML**: (Optional) Simple HTML report with collapsible sections.

## 6. Integration & CLI
- [x] **CLI Command construction**
    - Flags: `--path` (source code), `--schema` (SQL file), `--format` (output format).
- [x] **End-to-End Wiring**
    - Connect Scanner -> Extractor -> Parser(+Schema) -> Auditor -> Reporter.
