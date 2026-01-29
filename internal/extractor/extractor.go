package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"sql-check/internal/model"
	"strings"
)



// RegexExtractor is a basic extractor using regular expressions
type RegexExtractor struct {
}

func NewRegexExtractor() *RegexExtractor {
	return &RegexExtractor{}
}

// Patterns for different quote types
// Note: We use (?s) to allow . to match newlines for multi-line support
var (
	doubleQuoteSQL = regexp.MustCompile(`(?s)"(?i)(?:SELECT|INSERT|UPDATE|DELETE)\b.*?"`)
	singleQuoteSQL = regexp.MustCompile(`(?s)'(?i)(?:SELECT|INSERT|UPDATE|DELETE)\b.*?'`)
	backTickSQL    = regexp.MustCompile("(?s)`(?i)(?:SELECT|INSERT|UPDATE|DELETE)\\b.*?`")
)

func (e *RegexExtractor) Extract(filePath string, content []byte) ([]model.SQLSegment, error) {
	var segments []model.SQLSegment
	
	// Convert content to string once (allocating) is okay for now
	text := string(content)

	// Pre-calculate line offsets for fast line number lookup
	lineOffsets := make([]int, 0)
	for i, b := range text {
		if b == '\n' {
			lineOffsets = append(lineOffsets, i)
		}
	}

	getLineNo := func(idx int) int {
		// Binary search or linear scan. Linear is fine for now.
		line := 1
		for _, offset := range lineOffsets {
			if offset < idx {
				line++
			} else {
				break
			}
		}
		return line
	}

	for _, re := range []*regexp.Regexp{doubleQuoteSQL, singleQuoteSQL, backTickSQL} {
		// FindStringIndex returns [start, end] byte offsets
		matches := re.FindAllStringIndex(text, -1)
		for _, matchPos := range matches {
			start, end := matchPos[0], matchPos[1]
			matchedStr := text[start:end]
			
			if len(matchedStr) >= 2 {
				// Strip quotes
				sqlContent := matchedStr[1 : len(matchedStr)-1]
				
				segments = append(segments, model.SQLSegment{
					SQL: sqlContent,
					Location: model.Location{
						FilePath: filePath,
						Line:     getLineNo(start),
					},
					Language: "detected",
				})
			}
		}
	}
	
	return segments, nil
}


// Manager selects the appropriate extractor based on file extension
type Manager struct {
	extractors map[string]model.Extractor
}

func NewManager() *Manager {
	return &Manager{
		extractors: make(map[string]model.Extractor),
	}
}

func (m *Manager) Register(ext string, extr model.Extractor) {
	m.extractors[strings.ToLower(ext)] = extr
}

func (m *Manager) Extract(filePath string) ([]model.SQLSegment, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filePath), "."))
	if extr, ok := m.extractors[ext]; ok {
		return extr.Extract(filePath, content)
	}
	
	// Fallback or skip?
	// For now, fallback to generic regex if not found, or maybe just skip
	return NewRegexExtractor().Extract(filePath, content)
}
