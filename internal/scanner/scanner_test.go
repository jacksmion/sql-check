package scanner

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"sql-check/internal/model"
	"testing"
)

func TestFileWalker_Walk(t *testing.T) {
	// Create temp directory structure
	rootDir, err := os.MkdirTemp("", "scanner-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(rootDir)

	// Create files
	files := []string{
		"main.go",
		"main.py",
		"test.js",
		"ignored.txt",
		"sub/sub.go",
		"sub/ignore_dir/file.go",
		"vendor/vendor.go",
	}

	for _, f := range files {
		path := filepath.Join(rootDir, f)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package main"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name     string
		exts     []string
		excludes []string
		want     []string
	}{
		{
			name:     "Find Go files",
			exts:     []string{"go"},
			excludes: []string{"vendor", "ignore_dir"},
			want: []string{
				filepath.Join(rootDir, "main.go"),
				filepath.Join(rootDir, "sub/sub.go"),
			},
		},
		{
			name:     "Find Go and Py files",
			exts:     []string{"go", "py"},
			excludes: []string{"vendor", "ignore_dir"},
			want: []string{
				filepath.Join(rootDir, "main.go"),
				filepath.Join(rootDir, "main.py"),
				filepath.Join(rootDir, "sub/sub.go"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			walker := NewFileWalker(tt.exts, tt.excludes)
			
			// Collect results
			var got []string
			ctx := context.Background()
			paths, _ := walker.Walk(ctx, rootDir)

			// Reader loop
			done := make(chan struct{})
			go func() {
				defer close(done)
				for p := range paths {
					got = append(got, p)
				}
			}()
			<-done

			// Debugging info if it fails
			if len(got) != len(tt.want) {
				t.Logf("DEBUG: rootDir=%s", rootDir)
				filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
					t.Logf("DEBUG: Visiting %s (isDir=%v)", path, d.IsDir())
					return nil
				})
			}

			// Convert got paths to relative paths for comparison (normalized with ToSlash)
			var gotRel []string
			for _, p := range got {
				rel, err := filepath.Rel(rootDir, p)
				if err != nil {
					t.Fatalf("Rel error: %v", err)
				}
				gotRel = append(gotRel, filepath.ToSlash(rel))
			}

			// Expected relative paths
			var wantRel []string
			for _, p := range tt.want {
				rel, err := filepath.Rel(rootDir, p)
				if err != nil {
					t.Fatalf("Rel error: %v", err)
				}
				wantRel = append(wantRel, filepath.ToSlash(rel))
			}

			sort.Strings(gotRel)
			sort.Strings(wantRel)

			if !reflect.DeepEqual(gotRel, wantRel) {
				t.Errorf("%s: Walk() got %v, want %v", tt.name, gotRel, wantRel)
			}
		})
	}
}

func TestWorkerPool_Start(t *testing.T) {
	// Mock processor
	mockProc := func(path string) ([]model.SQLSegment, error) {
		return []model.SQLSegment{{SQL: "SELECT 1"}}, nil
	}

	pool := NewWorkerPool(2, mockProc)
	paths := make(chan string, 5)
	
	for i := 0; i < 5; i++ {
		paths <- "dummy_path"
	}
	close(paths)

	results := pool.Start(context.Background(), paths)

	count := 0
	for res := range results {
		if res.Error != nil {
			t.Errorf("WorkerPool error: %v", res.Error)
		}
		if len(res.Segments) != 1 {
			t.Errorf("Expected 1 segment, got %d", len(res.Segments))
		}
		count++
	}

	if count != 5 {
		t.Errorf("Expected 5 results, got %d", count)
	}
}
