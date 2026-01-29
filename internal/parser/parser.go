package parser

import (
	"fmt"
	"os"

	"sql-check/internal/model"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	_ "github.com/pingcap/tidb/parser/test_driver"
)

// SQLParser wraps the TiDB parser
type SQLParser struct {
	p *parser.Parser
}

func NewSQLParser() *SQLParser {
	return &SQLParser{
		p: parser.New(),
	}
}

// Parse converts a SQL string into an AST
func (sp *SQLParser) Parse(sql string) (ast.StmtNode, error) {
	// TiDB parser requires a semicolon or EOF. 
	// Often extracted fragments might not have it, or might have multiple.
	// We wrap it in a simple check.
	stmtNodes, _, err := sp.p.Parse(sql, "", "")
	if err != nil {
		return nil, err
	}
	if len(stmtNodes) == 0 {
		return nil, fmt.Errorf("no valid SQL found")
	}
	// For now, we return the first statement found
	return stmtNodes[0], nil
}

// LoadSchema reads a SQL file and populates the SchemaCtx
func (sp *SQLParser) LoadSchema(path string) (*model.SchemaCtx, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	schema := &model.SchemaCtx{
		Tables: make(map[string]*model.Table),
	}

	// Parse the whole schema file
	// Note: Parse returns []ast.StmtNode
	stmts, _, err := sp.p.Parse(string(content), "", "")
	if err != nil {
		return nil, fmt.Errorf("schema parse error: %w", err)
	}

	for _, stmt := range stmts {
		if createTable, ok := stmt.(*ast.CreateTableStmt); ok {
			table := parseCreateTable(createTable)
			schema.Tables[table.Name] = table
		}
	}

	return schema, nil
}

func parseCreateTable(node *ast.CreateTableStmt) *model.Table {
	t := &model.Table{
		Name:    node.Table.Name.O,
		Columns: make(map[string]*model.Column),
		Indexes: make([]*model.Index, 0),
	}

	// 1. Columns
	for _, col := range node.Cols {
		t.Columns[col.Name.Name.O] = &model.Column{
			Name: col.Name.Name.O,
			Type: col.Tp.String(), // Simplified type
		}
	}

	// 2. Constraints (PK, Unique, etc defined inline or at bottom)
	for _, cons := range node.Constraints {
		switch cons.Tp {
		case ast.ConstraintPrimaryKey, ast.ConstraintKey, ast.ConstraintIndex, ast.ConstraintUniq:
			idx := &model.Index{
				Name:    cons.Name,
				Unique:  cons.Tp == ast.ConstraintPrimaryKey || cons.Tp == ast.ConstraintUniq,
				Columns: make([]string, 0),
			}
			if idx.Name == "" && cons.Tp == ast.ConstraintPrimaryKey {
				idx.Name = "PRIMARY"
			}
			// Extract columns
			for _, keyCol := range cons.Keys {
				idx.Columns = append(idx.Columns, keyCol.Column.Name.O)
			}
			t.Indexes = append(t.Indexes, idx)
		}
	}

	return t
}
