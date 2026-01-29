package auditor

import (
	"sql-check/internal/model"
	"sql-check/internal/parser"
	"testing"

	"github.com/pingcap/tidb/parser/ast"
)

// MockRule for testing Auditor
type MockRule struct {
	issues []model.Issue
}

func (m *MockRule) Name() string { return "mock_rule" }
func (m *MockRule) Check(seg *model.SQLSegment, node ast.StmtNode, schema *model.SchemaCtx) ([]model.Issue, error) {
	return m.issues, nil
}

func TestAuditor_Audit(t *testing.T) {
	p := parser.NewSQLParser()
	a := NewAuditor(nil, p)

	// Register a rule that returns an issue
	expectedIssue := model.Issue{
		Type:    "MOCK_ISSUE",
		Message: "Mock issue found",
	}
	a.Register(&MockRule{issues: []model.Issue{expectedIssue}})

	// Input segments
	segments := []model.SQLSegment{
		{
			SQL: "SELECT 1",
			Location: model.Location{
				FilePath: "test.go",
				Line:     10,
			},
		},
	}

	// Audit
	issues, err := a.Audit(segments)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	if len(issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(issues))
	} else {
		if issues[0].Type != expectedIssue.Type {
			t.Errorf("Expected issue type %s, got %s", expectedIssue.Type, issues[0].Type)
		}
	}
}

func TestAuditor_Audit_ParseError(t *testing.T) {
	p := parser.NewSQLParser()
	a := NewAuditor(nil, p)

	// A rule, even if registered, shouldn't run if parse fails (logic in Auditor.Audit)
	a.Register(&MockRule{issues: []model.Issue{{Type: "SHOULD_NOT_HAPPEN"}}})

	segments := []model.SQLSegment{
		{
			SQL: "INVALID SQL syntax", 
		},
	}

	issues, err := a.Audit(segments)
	if err != nil {
		t.Fatalf("Audit() error = %v", err)
	}

	if len(issues) != 0 {
		t.Errorf("Expected 0 issues for invalid SQL, got %d", len(issues))
	}
}
