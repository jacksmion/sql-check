package main

import (
	"context"
	"fmt"
	"os"
	"sql-check/internal/auditor"
	"sql-check/internal/extractor"
	"sql-check/internal/model"
	"sql-check/internal/parser"
	"sql-check/internal/reporter"
	"sql-check/internal/scanner"

	"github.com/spf13/cobra"
)


var (
	srcPath    string
	schemaPath string
	reportFmt  string
	outputFile string
	excludes   []string
)

var rootCmd = &cobra.Command{
	Use:   "sql-check",
	Short: "A static analysis tool for SQL slow queries",
	Long: `sql-check is a CLI tool that scans your code for SQL queries, 
parses them, and checks against a provided database schema for 
common performance pitfalls like missing indexes, full table scans, etc.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Scanning source: %s\n", srcPath)
		if len(excludes) > 0 {
			fmt.Printf("Excluding patterns: %v\n", excludes)
		}
		if schemaPath != "" {
			fmt.Printf("Using schema: %s\n", schemaPath)
		}
		fmt.Printf("Report format: %s\n", reportFmt)
		
		return runAnalysis()
	},
}

func init() {
	rootCmd.Flags().StringVarP(&srcPath, "src", "s", ".", "Path to source code to scan")
	rootCmd.Flags().StringVarP(&schemaPath, "schema", "S", "schema.sql", "Path to database schema SQL file")
	rootCmd.Flags().StringVarP(&reportFmt, "report", "r", "console", "Report format (console, html)")
	rootCmd.Flags().StringVarP(&outputFile, "out", "o", "", "Output file path (default: 'report.html' for html)")
	rootCmd.Flags().StringSliceVarP(&excludes, "exclude", "e", []string{".git", "vendor", "*_test.go"}, "Glob patterns to exclude from scan")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runAnalysis() error {
	// 0. Validate Inputs
	if _, err := os.Stat(srcPath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", srcPath)
	}

	// 1. Initialize Extractor Manager
	mgr := extractor.NewManager()
	// Register generic regex extractor for all supported types for now
	generic := extractor.NewRegexExtractor()
	mgr.Register("go", generic)
	mgr.Register("py", generic)
	mgr.Register("cpp", generic)
	
	// Initialize Parser & Schema
	sqlParser := parser.NewSQLParser()
	var schema *model.SchemaCtx
	if schemaPath != "" {
		// Check if schema file exists
		if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
			fmt.Printf("Warning: Schema file not found at %s. Proceeding without context-aware checks.\n", schemaPath)
		} else {
			var err error
			fmt.Printf("Loading schema from %s...\n", schemaPath)
			schema, err = sqlParser.LoadSchema(schemaPath)
			if err != nil {
				return fmt.Errorf("failed to load schema: %w", err)
			}
			fmt.Printf("Schema loaded. Found %d tables.\n", len(schema.Tables))
		}
	}

	// 2. Initialize Scanner
	walker := scanner.NewFileWalker([]string{"go", "py", "cpp", "sql"}, excludes)

	
	ctx := context.Background()
	paths, errChan := walker.Walk(ctx, srcPath)

	// 3. Start Worker Pool
	pool := scanner.NewWorkerPool(10, func(path string) ([]model.SQLSegment, error) {
		return mgr.Extract(path)
	})
	results := pool.Start(ctx, paths)

	// Collect segments
	var allSegments []model.SQLSegment
	go func() {
		for err := range errChan {
			fmt.Printf("Scanner Error: %v\n", err)
		}
	}()

	fmt.Printf("Scanning started on %s...\n", srcPath)
	for res := range results {
		if res.Error != nil {
			// fmt.Printf("Extract Error on %s: %v\n", res.File, res.Error) // Optional verbose logging
			continue
		}
		if len(res.Segments) > 0 {
			allSegments = append(allSegments, res.Segments...)
		}
	}
	fmt.Printf("Scan complete. Validating %d SQL segments...\n", len(allSegments))

	// 4. Audit
	// Ensure schema is not nil if not loaded (empty context)
	if schema == nil {
		schema = &model.SchemaCtx{Tables: map[string]*model.Table{}}
	}

	auditEngine := auditor.NewAuditor(schema, sqlParser)
	auditEngine.Register(&auditor.NoWhereRule{})
	auditEngine.Register(&auditor.SelectStarRule{})
	auditEngine.Register(&auditor.IndexMissRule{})
	auditEngine.Register(&auditor.ImplicitConversionRule{})
	auditEngine.Register(&auditor.DeepPaginationRule{Threshold: 5000})
	auditEngine.Register(&auditor.NegativeQueryRule{})

	issues, err := auditEngine.Audit(allSegments)
	if err != nil {
		return fmt.Errorf("audit failed: %w", err)
	}

	// 5. Report
	var rpt model.Reporter
	
	switch reportFmt {
	case "html":
		target := outputFile
		if target == "" {
			target = "report.html"
		}
		rpt = reporter.NewHTMLReporter(target)
	default:
		rpt = reporter.NewConsoleReporter()
	}

	if err := rpt.Report(issues); err != nil {
		return fmt.Errorf("reporting failed: %w", err)
	}

	return nil
}



