package extractor

import (
	"reflect"
	"testing"
)

func TestRegexExtractor_Extract(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:    "Double quoted SQL",
			content: `db.Exec("SELECT * FROM users")`,
			expected: []string{
				"SELECT * FROM users",
			},
		},
		{
			name:    "Single quoted SQL",
			content: `ctx.Select('INSERT INTO logs VALUES')`,
			expected: []string{
				"INSERT INTO logs VALUES",
			},
		},
		{
			name:    "Backtick quoted SQL",
			content: "`UPDATE users SET name='test'`",
			expected: []string{
				"UPDATE users SET name='test'",
			},
		},
		{
			name:    "Multi-line string (simulated single line for regex)",
			content: `db.Query("SELECT * FROM users WHERE id = 1")`,
			expected: []string{
				"SELECT * FROM users WHERE id = 1",
			},
		},
		{
			name:     "No SQL",
			content:  `fmt.Println("Hello world")`,
			expected: nil,
		},
		{
			name:    "Mixed quotes",
			content: `db.Exec("DELETE FROM users"); log.Info('SELECT * FROM logs')`,
			expected: []string{
				"DELETE FROM users",
				"SELECT * FROM logs",
			},
		},
	}

	extractor := NewRegexExtractor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments, err := extractor.Extract("test.go", []byte(tt.content))
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			var got []string
			for _, seg := range segments {
				got = append(got, seg.SQL)
			}

			if len(got) == 0 && len(tt.expected) == 0 {
				return
			}

			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Extract() got = %v, want %v", got, tt.expected)
			}
		})
	}
}
