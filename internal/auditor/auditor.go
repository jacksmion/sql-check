package auditor

import (
	"fmt"
	"sql-check/internal/model"
	"sql-check/internal/parser"
)

type Auditor struct {
	rules  []model.Rule
	schema *model.SchemaCtx
	parser *parser.SQLParser
}

func NewAuditor(schema *model.SchemaCtx, p *parser.SQLParser) *Auditor {
	return &Auditor{
		rules:  make([]model.Rule, 0),
		schema: schema,
		parser: p,
	}
}

func (a *Auditor) Register(rule model.Rule) {
	a.rules = append(a.rules, rule)
}

func (a *Auditor) Audit(segments []model.SQLSegment) ([]model.Issue, error) {
	var allIssues []model.Issue

	for _, seg := range segments {
		// 1. Parse SQL
		stmt, err := a.parser.Parse(seg.SQL)
		if err != nil {
			// Report parse error as a warning or info?
			// For now, let's skip or log.
			// Ideally add a "ParseError" issue.
			continue
		}

		// 2. Run Rules
		for _, rule := range a.rules {
			issues, err := rule.Check(&seg, stmt, a.schema)
			if err != nil {
				fmt.Printf("Error running rule %s: %v\n", rule.Name(), err)
				continue
			}
			if len(issues) > 0 {
				allIssues = append(allIssues, issues...)
			}
		}
	}

	return allIssues, nil
}
