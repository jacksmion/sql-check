# SQL-Check: Static SQL Analyzer

**sql-check** is a powerful static analysis tool designed to detect potential slow queries, SQL anti-patterns, and performance risks directly in your source code. It treats your codebase like a database log, auditing SQL statements before they even reach production.

## 🚀 Features

*   **Multi-Language Support**: Automatically extracts SQL from **Go**, **Python**, **C++**, and generic file types.
*   **Schema Awareness**: Loads your database schema (`.sql` DDL) to provide context-aware auditing (e.g., index usage checks).
*   **Advanced Extraction**: Intelligent regex-based extractor handles SQL inside double quotes, single quotes, and backticks.
*   **Deep Auditing**:
    *   ❌ **Fatal Risks**: Unsafe `UPDATE`/`DELETE` without `WHERE`.
    *   ⚠️ **Performance Warnings**: Index misses (leftmost prefix), implicit type conversions, deep pagination, negative queries (`!=`, `NOT IN`), and leading wildcards in `LIKE`.
    *   💡 **Best Practices**: Detects `SELECT *` usage.
*   **Rich Reporting**: Outputs beautiful console logs or detailed **HTML** reports.

## 📦 Installation

Prerequisites: Go 1.20+

```bash
# Clone the repository
git clone https://github.com/yourusername/sql-check.git
cd sql-check

# Build the binary
go build -o sql-check cmd/sql-check/main.go
```

## 🛠 Usage

### 1. Basic Scan
Scan the current directory for SQL issues:

```bash
./sql-check --src .
```

### 2. Context-Aware Scan (Recommended)
Provide a schema file to enable powerful index checking rule:

```bash
./sql-check --src ./backend --schema ./db/schema.sql
```

### 3. Generate HTML Report
Export the results to a shareable HTML file:

```bash
./sql-check --src . --schema schema.sql --report html --out audit-report.html
```

### 4. Filter Files
Exclude test files or specific folders:

```bash
./sql-check --src . --exclude "*_test.go" --exclude "migrations"
```

## ⚙️ Logic & Architecture

The tool operates in pipeline phases:

1.  **Scanner**: Concurrent file system walker (Producer-Consumer model).
2.  **Extractor**: regex-based engine identifies SQL strings in code.
3.  **Parser**: Uses `tidb/parser` to convert SQL text into Abstract Syntax Trees (AST).
4.  **Auditor**: Runs a suite of rules against the AST and loaded Schema.
    *   *IndexMissRule*: Checks if `WHERE` columns hit any table index.
    *   *ImplicitConversionRule*: Checks simple type mismatches (e.g., String col vs Int value).
5.  **Reporter**: Formats the findings.

## 🛡 Supported Rules

| Rule Name | Level | Description |
| :--- | :--- | :--- |
| `NO_WHERE_CLAUSE` | **FATAL** | `UPDATE` or `DELETE` with no condition (Full Table Write). |
| `INDEX_MISS` | **WARN** | Query condition does not hit any index prefix. |
| `IMPLICIT_CONVERSION` | **WARN** | Comparison between different types (triggers full scan). |
| `DEEP_PAGINATION` | **WARN** | `LIMIT offset, count` where offset > 5000. |
| `LEADING_WILDCARD` | **WARN** | `LIKE '%abc'` prevents index usage. |
| `NEGATIVE_QUERY` | **WARN** | Usage of `!=` or `NOT IN`. |
| `SELECT_STAR` | **SUGGESTION** | Usage of `SELECT *`. |

## 🤝 Contributing

Contributions are welcome! Please submit a Pull Request or open an Issue.

## 📄 License

MIT License
