package model

import (
	"github.com/pingcap/tidb/parser/ast"
)

// Extractor is responsible for parsing a file and finding SQL segments
type Extractor interface {
	// Extract parses the given file content and returns found SQL segments
	Extract(filePath string, content []byte) ([]SQLSegment, error)
}

// Rule represents a single audit logic unit
type Rule interface {
	// Name returns the unique identifier of the rule
	Name() string
	// Check examines the SQL segment and returns any issues found
	// It receives the SQL segment, the parsed AST, and the Schema context
	Check(segment *SQLSegment, node ast.StmtNode, schema *SchemaCtx) ([]Issue, error)
}


// Reporter defines how to output results
type Reporter interface {
	Report(issues []Issue) error
}
