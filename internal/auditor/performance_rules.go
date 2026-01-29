package auditor

import (
	"sql-check/internal/model"
	"strings"

	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
)

// DeepPaginationRule detects LIMIT offset, count where offset is large
type DeepPaginationRule struct {
	Threshold int64
}

func (r *DeepPaginationRule) Name() string { return "deep_pagination" }

func (r *DeepPaginationRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue
	limitThreshold := r.Threshold
	if limitThreshold == 0 {
		limitThreshold = 5000 // default
	}

	// Helper to check Limit node
	checkLimit := func(limit *ast.Limit) {
		if limit != nil && limit.Offset != nil {
			if val, ok := limit.Offset.(*test_driver.ValueExpr); ok {
				if intVal, ok := val.GetValue().(int64); ok && intVal > limitThreshold {
					issues = append(issues, model.Issue{
						Type:       "DEEP_PAGINATION",
						Level:      model.RiskLevelWarning,
						Message:    "Deep pagination detected (High Offset)",
						Suggestion: "Use keyset pagination (WHERE id > last_id) instead of OFFSET.",
						Segment:    *seg,
					})
				}
			}
		}
	}

	switch stmt := node.(type) {
	case *ast.SelectStmt:
		checkLimit(stmt.Limit)
	// case *ast.UnionStmt: // Removed as it might be named SetOprStmt or undefined
	// 	checkLimit(stmt.Limit)
	}

	return issues, nil
}

// NegativeQueryRule detects !=, NOT IN, LIKE '%...'
type NegativeQueryRule struct{}

func (r *NegativeQueryRule) Name() string { return "negative_query" }

func (r *NegativeQueryRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	var issues []model.Issue

	// We use a visitor to find expressions anywhere in the statement
	v := &negativeVisitor{issues: &issues, seg: seg}
	node.Accept(v)

	return issues, nil
}

type negativeVisitor struct {
	issues *[]model.Issue
	seg    *model.SQLSegment
}

func (v *negativeVisitor) Enter(in ast.Node) (ast.Node, bool) {
	if pattern, ok := in.(*ast.PatternInExpr); ok && pattern.Not {
		*v.issues = append(*v.issues, model.Issue{
			Type:       "NEGATIVE_QUERY",
			Level:      model.RiskLevelWarning,
			Message:    "Avoid using NOT IN",
			Suggestion: "Use NOT EXISTS or LEFT JOIN ... IS NULL which are often better optimized.",
			Segment:    *v.seg,
		})
	}

	if binOp, ok := in.(*ast.BinaryOperationExpr); ok {
		if binOp.Op == opcode.NE { // !=
			*v.issues = append(*v.issues, model.Issue{
				Type:       "NEGATIVE_QUERY",
				Level:      model.RiskLevelWarning,
				Message:    "Avoid using != (Not Equal)",
				Suggestion: "Negative comparison often prevents index usage.",
				Segment:    *v.seg,
			})
		}
	}
	
	if pattern, ok := in.(*ast.PatternLikeOrIlikeExpr); ok {
		// Check for leading wildcard
		if strVal, ok := pattern.Pattern.(*test_driver.ValueExpr); ok {
			s := strVal.GetString()
			if strings.HasPrefix(s, "%") {
				*v.issues = append(*v.issues, model.Issue{
					Type:       "LEADING_WILDCARD",
					Level:      model.RiskLevelWarning,
					Message:    "LIKE query with leading wildcard",
					Suggestion: "Leading wildcards confuse the optimizer and prevent index usage (Full Table Scan).",
					Segment:    *v.seg,
				})
			}
		}
	}

	return in, false
}


func (v *negativeVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}
