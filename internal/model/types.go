package model

import "fmt"

// Location represents the physical location of a code segment
type Location struct {
	FilePath string
	Line     int
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d", l.FilePath, l.Line)
}

// SQLSegment represents an extracted SQL statement from source code
type SQLSegment struct {
	SQL      string
	Location Location
	Language string // e.g., "go", "python", "cpp"
}

// RiskLevel defines the severity of an audit finding
type RiskLevel string

const (
	RiskLevelFatal      RiskLevel = "FATAL"
	RiskLevelWarning    RiskLevel = "WARNING"
	RiskLevelSuggestion RiskLevel = "SUGGESTION"
)

// Issue represents a potential problem found by the auditor
type Issue struct {
	Type        string    // e.g., "NO_WHERE_CLAUSE", "INDEX_MISSING"
	Level       RiskLevel
	Message     string
	Suggestion  string
	Segment     SQLSegment
}

// SchemaCtx represents the loaded database schema context
type SchemaCtx struct {
	Tables map[string]*Table
}

type Table struct {
	Name    string
	Columns map[string]*Column
	Indexes []*Index
}

type Column struct {
	Name string
	Type string // Simplified type representation
}

type Index struct {
	Name    string
	Columns []string // Ordered list of column names in the index
	Unique  bool
}
