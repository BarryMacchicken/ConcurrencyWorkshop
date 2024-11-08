package filefinder

import (
	"context"
	"os"
	"path/filepath"
	"sync"
)

type concurrent struct {
	FileFinder
}

func NewConcurrent() FileFinder {
	return &concurrent{}
}

func searchDir(ctx context.Context, rootPath, filename string, wg *sync.WaitGroup, fileChan chan string, errChan chan error) {
	defer wg.Done()
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		errChan <- err
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			if entry.Name() == filename {
				fileChan <- filepath.Join(rootPath, filename)
				return
			}
			continue
		}

		subPath := filepath.Join(rootPath, entry.Name())
		wg.Add(1)
		go searchDir(ctx, subPath, filename, wg, fileChan, errChan)
	}
}

func (s *concurrent) FindFile(ctx context.Context, rootPath, filename string) (string, error) {
	fileChan := make(chan string)
	errChan := make(chan error)
	wg := &sync.WaitGroup{}
	allEnded := false

	wg.Add(1)
	go searchDir(ctx, rootPath, filename, wg, fileChan, errChan)
	go func() {
		wg.Wait()
		close(fileChan)
		close(errChan)
		allEnded = true
	}()

	for {
		select {
		case file := <-fileChan:
			if file != "" {
				return file, nil
			}
			if allEnded && file == "" {
				return "", ErrNotFound
			}
		case err := <-errChan:
			if err != nil {
				return "", err
			}
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}
