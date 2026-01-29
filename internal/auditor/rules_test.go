package auditor

import (
	"sql-check/internal/model"
	"sql-check/internal/parser"
	"testing"
)

func TestNoWhereRule_Check(t *testing.T) {
	p := parser.NewSQLParser()
	rule := &NoWhereRule{}

	tests := []struct {
		name       string
		sql        string
		wantIssues int
	}{
		{
			name:       "UPDATE without WHERE",
			sql:        "UPDATE users SET name = 'test'",
			wantIssues: 1,
		},
		{
			name:       "UPDATE with WHERE",
			sql:        "UPDATE users SET name = 'test' WHERE id = 1",
			wantIssues: 0,
		},
		{
			name:       "DELETE without WHERE",
			sql:        "DELETE FROM users",
			wantIssues: 1,
		},
		{
			name:       "DELETE with WHERE",
			sql:        "DELETE FROM users WHERE id = 1",
			wantIssues: 0,
		},
		{
			name:       "SELECT ignored",
			sql:        "SELECT * FROM users",
			wantIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := p.Parse(tt.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			seg := &model.SQLSegment{SQL: tt.sql}
			
			issues, err := rule.Check(seg, stmt, nil)
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}

			if len(issues) != tt.wantIssues {
				t.Errorf("Check() got %d issues, want %d", len(issues), tt.wantIssues)
			}
		})
	}
}

func TestSelectStarRule_Check(t *testing.T) {
	p := parser.NewSQLParser()
	rule := &SelectStarRule{}

	tests := []struct {
		name       string
		sql        string
		wantIssues int
	}{
		{
			name:       "SELECT *",
			sql:        "SELECT * FROM users",
			wantIssues: 1,
		},
		{
			name:       "SELECT columns",
			sql:        "SELECT id, name FROM users",
			wantIssues: 0,
		},
		{
			name:       "SELECT * with aggregate",
			sql:        "SELECT count(*) FROM users",
			wantIssues: 0, // count(*) is usually fine or different rule
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := p.Parse(tt.sql)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			seg := &model.SQLSegment{SQL: tt.sql}

			issues, err := rule.Check(seg, stmt, nil)
			if err != nil {
				t.Fatalf("Check failed: %v", err)
			}

			if len(issues) != tt.wantIssues {
				// Special check for count(*) which might be parsed differently
				// TiDB parser parses count(*) as an aggregate function, not wildcard field usually.
				// But let's verify behavior.
				// Actually SELECT * is wildcard field.
				t.Errorf("Check() got %d issues, want %d", len(issues), tt.wantIssues)
			}
		})
	}
}
