package extractor

import (
	"bufio"
	"bytes"
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
// Note: We use non-greedy *? to stop at the first closing quote
// We can't use backreferences in Go regexp (RE2)
var (
	doubleQuoteSQL = regexp.MustCompile(`"(?i)(?:SELECT|INSERT|UPDATE|DELETE)\b.*?"`)
	singleQuoteSQL = regexp.MustCompile(`'(?i)(?:SELECT|INSERT|UPDATE|DELETE)\b.*?'`)
	backTickSQL    = regexp.MustCompile("`(?i)(?:SELECT|INSERT|UPDATE|DELETE)\\b.*?`")
)

func (e *RegexExtractor) Extract(filePath string, content []byte) ([]model.SQLSegment, error) {
	var segments []model.SQLSegment
	
	scanner := bufio.NewScanner(bytes.NewReader(content))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()

		for _, re := range []*regexp.Regexp{doubleQuoteSQL, singleQuoteSQL, backTickSQL} {
			matches := re.FindAllString(line, -1)
			for _, match := range matches {
				if len(match) >= 2 {
					// Strip quotes
					sqlContent := match[1 : len(match)-1]
					segments = append(segments, model.SQLSegment{
						SQL: sqlContent,
						Location: model.Location{
							FilePath: filePath,
							Line:     lineNo,
						},
						Language: "detected",
					})
				}
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
