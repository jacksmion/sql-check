package auditor

import (
	"fmt"
	"sql-check/internal/model"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/test_driver"
)

// ImplicitConversionRule detects type mismatches between columns and values
type ImplicitConversionRule struct{}

func (r *ImplicitConversionRule) Name() string { return "implicit_conversion" }

func (r *ImplicitConversionRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	// Find the table name first (reusing logic from IndexMissRule would be better, but duplication for now is safer)
	var tableName string
	switch stmt := node.(type) {
	case *ast.SelectStmt:
		if stmt.From != nil {
			if ts, ok := stmt.From.TableRefs.Left.(*ast.TableSource); ok {
				if tn, ok := ts.Source.(*ast.TableName); ok {
					tableName = tn.Name.O
				}
			}
		}
	}
	// Also support Update/Delete if needed, focused on Select for now

	if tableName == "" {
		return nil, nil
	}

	table, ok := schema.Tables[tableName]
	if !ok {
		return nil, nil
	}

	v := &typeVisitor{
		issues:    &issues,
		seg:       seg,
		table:     table,
		columns:   table.Columns,
	}
	node.Accept(v)

	return issues, nil
}

type typeVisitor struct {
	issues  *[]model.Issue
	seg     *model.SQLSegment
	table   *model.Table
	columns map[string]*model.Column
}

func (v *typeVisitor) Enter(in ast.Node) (ast.Node, bool) {
	if binOp, ok := in.(*ast.BinaryOperationExpr); ok {
		// Check for Col = Value or Value = Col
		lCol, lOk := binOp.L.(*ast.ColumnNameExpr)
		rVal, rOk := binOp.R.(*test_driver.ValueExpr)
		
		if lOk && rOk {
			v.checkMismatch(lCol.Name.Name.O, rVal)
		} else {
			lVal, lOk := binOp.L.(*test_driver.ValueExpr)
			rCol, rOk := binOp.R.(*ast.ColumnNameExpr)
			if lOk && rOk {
				v.checkMismatch(rCol.Name.Name.O, lVal)
			}
		}
	}
	return in, false
}

func (v *typeVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

func (v *typeVisitor) checkMismatch(colName string, valExpr *test_driver.ValueExpr) {
	colDef, ok := v.columns[colName]
	if !ok {
		return
	}

	colType := strings.ToUpper(colDef.Type)
	isStringCol := strings.Contains(colType, "CHAR") || strings.Contains(colType, "TEXT")
	
	// ValueExpr Kind inspection
	val := valExpr.GetValue()
	
	// Check: String Column compared with Int Value
	if isStringCol {
		switch val.(type) {
		case int, int64, float64:
			*v.issues = append(*v.issues, model.Issue{
				Type:       "IMPLICIT_CONVERSION",
				Level:      model.RiskLevelWarning,
				Message:    fmt.Sprintf("Explicit implicit conversion detected: String column '%s' compared with Number.", colName),
				Suggestion: "Quote the number to avoid implicit conversion and index invalidation (e.g., '123' instead of 123).",
				Segment:    *v.seg,
			})
		}
	}
}
