package parser

import (
	"github.com/pingcap/tidb/parser/ast"
)

// ExtractTableNames extracts all table names mentioned in a SQL statement.
// Currently supports Select, Update, and Delete statements.
func ExtractTableNames(node ast.StmtNode) []string {
	var tables []string
	
	switch stmt := node.(type) {
	case *ast.SelectStmt:
		if stmt.From != nil {
			extractTableRefs(stmt.From.TableRefs, &tables)
		}
	case *ast.UpdateStmt:
		if stmt.TableRefs != nil && stmt.TableRefs.TableRefs != nil {
			extractTableRefs(stmt.TableRefs.TableRefs, &tables)
		}
	case *ast.DeleteStmt:
		if stmt.TableRefs != nil && stmt.TableRefs.TableRefs != nil {
			extractTableRefs(stmt.TableRefs.TableRefs, &tables)
		}
	case *ast.InsertStmt:
		if stmt.Table != nil {
			extractTableRefs(stmt.Table.TableRefs, &tables)
		}
	}

	return tables
}

func extractTableRefs(join *ast.Join, tables *[]string) {
	if join == nil {
		return
	}
	
	if join.Left != nil {
		extractTableSource(join.Left, tables)
	}
	if join.Right != nil {
		extractTableSource(join.Right, tables)
	}
}

func extractTableSource(r ast.ResultSetNode, tables *[]string) {
	if ts, ok := r.(*ast.TableSource); ok {
		if tn, ok := ts.Source.(*ast.TableName); ok {
			*tables = append(*tables, tn.Name.O)
		}
	} else if join, ok := r.(*ast.Join); ok {
		extractTableRefs(join, tables)
	}
}
