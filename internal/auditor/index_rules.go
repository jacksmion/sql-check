package auditor

import (
	"fmt"
	"sql-check/internal/model"
	"sql-check/internal/parser"

	"github.com/pingcap/tidb/parser/ast"
)

// IndexMissRule checks if WHERE usage aligns with available indexes
type IndexMissRule struct{}

func (r *IndexMissRule) Name() string { return "index_miss" }

func (r *IndexMissRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	// 1. Identify Target Table Name and WHERE clause
	// 1. Identify Target Table Name and WHERE clause
	tables := parser.ExtractTableNames(node)
	if len(tables) == 0 {
		return nil, nil
	}
	tableName := tables[0] // Simplify: check strictly the first table found

	var whereExpr ast.ExprNode
	switch stmt := node.(type) {
	case *ast.SelectStmt:
		whereExpr = stmt.Where
	case *ast.UpdateStmt:
		whereExpr = stmt.Where
	case *ast.DeleteStmt:
		whereExpr = stmt.Where
	}

	if tableName == "" || whereExpr == nil {
		return nil, nil // Nothing to check or complex query
	}

	// 2. Lookup Table in Schema
	table, ok := schema.Tables[tableName]
	if !ok {
		// Table not found in schema, maybe alias or missing schema
		return nil, nil
	}

	// 3. Extract Columns used in WHERE as simple Equality or Range
	// We only care about columns that are candidates for indexing (e.g. A=1, A IN (..), A > 1)
	usedCols := make(map[string]bool)
	v := &columnVisitor{cols: usedCols}
	whereExpr.Accept(v)

	if len(usedCols) == 0 {
		return nil, nil // No columns found in where? strange
	}

	// 4. Check against Indexes
	// Strategy: At least ONE index must have its FIRST column present in usedCols.
	// This ensures we are not doing a full table scan (usually).
	
	hasHit := false

	if len(table.Indexes) == 0 {
		// Only primary key? The parser might not separate PK from indexes in my simplistic loader if implicit.
		// My loader adds PK to indexes list, so this is fine.
		issues = append(issues, model.Issue{
			Type:       "NO_INDEXES_DEFINED",
			Level:      model.RiskLevelWarning,
			Message:    fmt.Sprintf("Table '%s' has no indexes defined.", tableName),
			Suggestion: "Add indexes to optimize queries.",
			Segment:    *seg,
		})
		return issues, nil
	}

	for _, idx := range table.Indexes {
		if len(idx.Columns) > 0 {
			firstCol := idx.Columns[0]
			if usedCols[firstCol] {
				hasHit = true
				break
			}
		}
	}

	if !hasHit {
		// Construct error message with available indexes
		var indexStr string
		for _, idx := range table.Indexes {
			indexStr += fmt.Sprintf("[%s(%v)] ", idx.Name, idx.Columns)
		}
		
		issues = append(issues, model.Issue{
			Type:       "INDEX_MISS",
			Level:      model.RiskLevelWarning,
			Message:    fmt.Sprintf("Query on '%s' does not hit any index prefix. WHERE uses %v but available indexes are: %s", tableName, mapKeys(usedCols), indexStr),
			Suggestion: "Ensure the WHERE clause filters on the leftmost column of an index.",
			Segment:    *seg,
		})
	}

	return issues, nil
}

func mapKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

type columnVisitor struct {
	cols map[string]bool
}

func (v *columnVisitor) Enter(in ast.Node) (ast.Node, bool) {
	if funcCall, ok := in.(*ast.FuncCallExpr); ok {
		// Traverse function arguments to mark columns as used inside function
		for _, arg := range funcCall.Args {
			checkForColumn(arg, v.cols, true)
		}
		return in, true // Skip children as we handled them manually
	}

	if col, ok := in.(*ast.ColumnName); ok {
		v.cols[col.Name.O] = false // false means "clean usage" (not in function)
	}
	
	return in, false
}

func (v *columnVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func checkForColumn(node ast.Node, cols map[string]bool, inFunc bool) {
	if col, ok := node.(*ast.ColumnName); ok {
		if !inFunc {
			cols[col.Name.O] = true
		}
	} else {
		// Manual recursion for common expression types used in args
		// This is a simplified traversal since we don't have the visitor here
		switch expr := node.(type) {
		case *ast.BinaryOperationExpr:
			checkForColumn(expr.L, cols, inFunc)
			checkForColumn(expr.R, cols, inFunc)
		case *ast.ParenthesesExpr:
			checkForColumn(expr.Expr, cols, inFunc)
		case *ast.FuncCallExpr:
			for _, arg := range expr.Args {
				checkForColumn(arg, cols, true) // Nested function is still inFunc=true
			}
		// Add other types as needed
		}
	}
}
