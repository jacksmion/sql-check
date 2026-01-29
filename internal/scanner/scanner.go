package scanner

import (
	"context"
	"io/fs"
	"path/filepath"
	"sql-check/internal/model"
	"strings"
	"sync"
)

// FileWalker is responsible for traversing directories and feeding files to a channel
type FileWalker struct {
	Extensions map[string]struct{}
	Excludes   []string
}

func NewFileWalker(exts []string, excludes []string) *FileWalker {
	e := make(map[string]struct{})
	for _, ext := range exts {
		e[strings.ToLower(ext)] = struct{}{}
	}
	return &FileWalker{
		Extensions: e,
		Excludes:   excludes,
	}
}

// Walk starts the traversal and returns a channel of file paths.
// It runs in a separate goroutine and closes the channel when done.
func (fw *FileWalker) Walk(ctx context.Context, root string) (<-chan string, <-chan error) {
	paths := make(chan string, 100) // Buffered channel
	errs := make(chan error, 1)

	go func() {
		defer close(paths)
		defer close(errs)

		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// Check cancellation
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// TODO: Add .gitignore support here

			if d.IsDir() {
				// Check for exclusions (Simple containment or glob)
				for _, exclude := range fw.Excludes {
					if strings.Contains(path, exclude) {
						return filepath.SkipDir
					}
				}
				if strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
					return filepath.SkipDir // Skip hidden directories like .git
				}
				return nil
			}

			// File filtering logic
			// Check exclusions for files (e.g. *_test.go)
			for _, exclude := range fw.Excludes {
				matched, _ := filepath.Match(exclude, d.Name())
				if matched || strings.Contains(path, exclude) {
					return nil // Skip this file
				}
			}


			// Check extension
			ext := strings.ToLower(filepath.Ext(path))
			if len(ext) > 0 {
				ext = ext[1:] // remove dot
			}
			if _, ok := fw.Extensions[ext]; ok {
				select {
				case paths <- path:
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			return nil
		})

		if err != nil {
			errs <- err
		}
	}()

	return paths, errs
}

type ScanResult struct {
	File     string
	Segments []model.SQLSegment
	Error    error
}

// Processor defines a function that processes a file
type Processor func(path string) ([]model.SQLSegment, error)

// WorkerPool manages concurrent processing
type WorkerPool struct {
	Concurrency int
	Processor   Processor
}

func NewWorkerPool(concurrency int, proc Processor) *WorkerPool {
	return &WorkerPool{
		Concurrency: concurrency,
		Processor:   proc,
	}
}

func (wp *WorkerPool) Start(ctx context.Context, paths <-chan string) <-chan ScanResult {
	results := make(chan ScanResult)
	var wg sync.WaitGroup

	for i := 0; i < wp.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range paths {
				select {
				case <-ctx.Done():
					return
				default:
					res, err := wp.Processor(path)
					// We send result even if err is present, to report extraction errors
					select {
					case results <- ScanResult{File: path, Segments: res, Error: err}:
					case <-ctx.Done():
						return
					}
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

