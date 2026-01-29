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

			// Get relative path
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if rel == "." {
				return nil
			}

			// Normalize and split
			rel = filepath.ToSlash(rel)
			parts := strings.Split(rel, "/")

			// Check if any part of the path is excluded or hidden
			for _, part := range parts {
				// Skip hidden
				if strings.HasPrefix(part, ".") && part != "." {
					if d.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
				// Skip excluded
				for _, exclude := range fw.Excludes {
					if part == exclude {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					if matched, _ := filepath.Match(exclude, part); matched {
						if d.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
				}
			}

			if d.IsDir() {
				return nil
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

