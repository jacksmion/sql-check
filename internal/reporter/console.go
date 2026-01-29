package reporter

import (
	"fmt"
	"io"
	"os"
	"sql-check/internal/model"

	"github.com/fatih/color"
)

type ConsoleReporter struct {
	out io.Writer
}

func NewConsoleReporter() *ConsoleReporter {
	return &ConsoleReporter{out: os.Stdout}
}

func (r *ConsoleReporter) Report(issues []model.Issue) error {
	if len(issues) == 0 {
		fmt.Fprintln(r.out, color.GreenString("✔ No SQL issues found! Great job."))
		return nil
	}

	for _, issue := range issues {
		// Format: file:line: [LEVEL] Message
		loc := fmt.Sprintf("%s:%d", issue.Segment.Location.FilePath, issue.Segment.Location.Line)
		
		var levelColor *color.Color
		switch issue.Level {
		case model.RiskLevelFatal:
			levelColor = color.New(color.FgRed, color.Bold)
		case model.RiskLevelWarning:
			levelColor = color.New(color.FgYellow, color.Bold)
		case model.RiskLevelSuggestion:
			levelColor = color.New(color.FgBlue, color.Bold)
		default:
			levelColor = color.New(color.FgWhite)
		}

		fmt.Fprintf(r.out, "%s: [%s] %s\n", loc, levelColor.Sprint(issue.Level), issue.Message)
		
		// Print code snippet context if possible (simplified here)
		fmt.Fprintf(r.out, "\tCode: %s\n", color.CyanString(truncate(issue.Segment.SQL, 80)))
		fmt.Fprintf(r.out, "\tSuggestion: %s\n", issue.Suggestion)
		fmt.Fprintln(r.out)
	}
	
	// Summary
	fmt.Fprintf(r.out, "\n%s found %d issues.\n", color.RedString("✘"), len(issues))
	return nil
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
