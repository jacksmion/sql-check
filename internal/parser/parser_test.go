package parser

import (
	"os"
	"testing"
)

func TestSQLParser_Parse(t *testing.T) {
	parser := NewSQLParser()

	tests := []struct {
		name    string
		sql     string
		wantErr bool
	}{
		{
			name:    "Valid SELECT",
			sql:     "SELECT * FROM users",
			wantErr: false,
		},
		{
			name:    "Valid INSERT",
			sql:     "INSERT INTO users (name) VALUES ('test')",
			wantErr: false,
		},
		{
			name:    "Invalid SQL",
			sql:     "SELECT * FROM",
			wantErr: true,
		},
		{
			name:    "Empty SQL",
			sql:     "", // TiDB parser might return text as empty
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stmt, err := parser.Parse(tt.sql)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && stmt == nil {
				t.Errorf("Parse() returned nil statement for valid SQL")
			}
		})
	}
}

func TestSQLParser_LoadSchema(t *testing.T) {
	// Create a temporary schema file
	content := `
		CREATE TABLE users (
			id INT PRIMARY KEY,
			name VARCHAR(255),
			email VARCHAR(255),
			KEY idx_email (email)
		);
	`
	tmpfile, err := os.CreateTemp("", "schema-*.sql")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	parser := NewSQLParser()
	schema, err := parser.LoadSchema(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadSchema() error = %v", err)
	}

	// Verify schema content
	if len(schema.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(schema.Tables))
	}

	table, ok := schema.Tables["users"]
	if !ok {
		t.Fatalf("Table 'users' not found")
	}

	if len(table.Columns) != 3 {
		t.Errorf("Expected 3 columns, got %d", len(table.Columns))
	}

	if len(table.Indexes) != 2 { // One PK + One KEY
		t.Errorf("Expected 2 indexes, got %d", len(table.Indexes))
	}
}
