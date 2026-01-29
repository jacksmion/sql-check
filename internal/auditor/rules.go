package auditor

import (
	"sql-check/internal/model"

	"github.com/pingcap/tidb/parser/ast"
)

// NoWhereRule detects UPDATE/DELETE without WHERE
type NoWhereRule struct{}

func (r *NoWhereRule) Name() string { return "no_where_clause" }

func (r *NoWhereRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	switch stmt := node.(type) {
	case *ast.UpdateStmt:
		if stmt.Where == nil {
			issues = append(issues, model.Issue{
				Type:       "UNSAFE_UPDATE",
				Level:      model.RiskLevelFatal,
				Message:    "UPDATE statement executed without WHERE clause (Full Table Update)",
				Suggestion: "Add a WHERE clause to limit the scope of the update.",
				Segment:    *seg,
			})
		}
	case *ast.DeleteStmt:
		if stmt.Where == nil {
			issues = append(issues, model.Issue{
				Type:       "UNSAFE_DELETE",
				Level:      model.RiskLevelFatal,
				Message:    "DELETE statement executed without WHERE clause (Full Table Delete)",
				Suggestion: "Add a WHERE clause to limit the scope of the delete.",
				Segment:    *seg,
			})
		}
	}

	return issues, nil
}

// SelectStarRule detects SELECT *
type SelectStarRule struct{}

func (r *SelectStarRule) Name() string { return "select_star" }

func (r *SelectStarRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	if stmt, ok := node.(*ast.SelectStmt); ok {
		for _, field := range stmt.Fields.Fields {
			if field.WildCard != nil {
				issues = append(issues, model.Issue{
					Type:       "SELECT_STAR",
					Level:      model.RiskLevelSuggestion,
					Message:    "Avoid using SELECT * in production",
					Suggestion: "List valid columns explicitly to reduce I/O and forward compatibility issues.",
					Segment:    *seg,
				})
			}
		}
	}

	return issues, nil
}
