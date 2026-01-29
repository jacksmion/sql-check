package auditor

import (
	"fmt"
	"sql-check/internal/model"

	"github.com/pingcap/tidb/parser/ast"
)

// IndexMissRule checks if WHERE usage aligns with available indexes
type IndexMissRule struct{}

func (r *IndexMissRule) Name() string { return "index_miss" }

func (r *IndexMissRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	// 1. Identify Target Table Name and WHERE clause
	var tableName string
	var whereExpr ast.ExprNode

	switch stmt := node.(type) {
	case *ast.SelectStmt:
		if stmt.From != nil && len(stmt.From.TableRefs.Left.(*ast.TableSource).Source.(*ast.TableName).Name.O) > 0 {
			// Very simplified: takes the first table. JOINs are harder.
			// Ideally we traverse Join nodes.
			if ts, ok := stmt.From.TableRefs.Left.(*ast.TableSource); ok {
				if tn, ok := ts.Source.(*ast.TableName); ok {
					tableName = tn.Name.O
				}
			}
		}
		whereExpr = stmt.Where
	case *ast.UpdateStmt:
		if stmt.TableRefs != nil && stmt.TableRefs.TableRefs != nil {
			if ts, ok := stmt.TableRefs.TableRefs.Left.(*ast.TableSource); ok {
				if tn, ok := ts.Source.(*ast.TableName); ok {
					tableName = tn.Name.O
				}
			}
		}
		whereExpr = stmt.Where
	case *ast.DeleteStmt:
		if stmt.TableRefs != nil && stmt.TableRefs.TableRefs != nil {
			if ts, ok := stmt.TableRefs.TableRefs.Left.(*ast.TableSource); ok {
				if tn, ok := ts.Source.(*ast.TableName); ok {
					tableName = tn.Name.O
				}
			}
		}
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
	if col, ok := in.(*ast.ColumnName); ok {
		v.cols[col.Name.O] = true
	}
	
	// Handle cases like "WHERE function(col)" -> This usually invalidates the index usage for that col
	// But our simplified logic just checks if col appears. 
	// To be stricter, we should check parent nodes.
	// For now, simple "is present" is good enough for a basic "Leftmost Prefix" check.
	// Refinement: If inside FuncCallExpr, we might want to ignore it or flag it?
	// Leaving simple for now.

	return in, false
}

func (v *columnVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
