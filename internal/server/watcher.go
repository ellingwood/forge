package server

import (
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors filesystem paths for changes and invokes a callback
// function when modifications are detected. It uses debouncing to coalesce
// rapid successive changes into a single callback invocation.
type Watcher struct {
	paths    []string
	onChange func()
	debounce time.Duration
	watcher  *fsnotify.Watcher
	done     chan struct{}
	once     sync.Once
}

// NewWatcher creates a new Watcher that monitors the given paths for changes.
// The onChange callback is invoked after changes have been debounced for the
// specified duration.
func NewWatcher(paths []string, debounce time.Duration, onChange func()) *Watcher {
	return &Watcher{
		paths:    paths,
		onChange: onChange,
		debounce: debounce,
		done:     make(chan struct{}),
	}
}

// Start begins watching the configured paths for changes. It blocks until
// Stop is called or a fatal error occurs.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsw

	// Add paths to the watcher. For directories, recursively add
	// subdirectories as fsnotify does not watch recursively by default.
	for _, p := range w.paths {
		info, err := os.Stat(p)
		if err != nil {
			// Path may not exist (e.g. no assets/ directory); skip.
			continue
		}
		if info.IsDir() {
			if err := w.addRecursive(p); err != nil {
				log.Printf("warning: failed to watch %s: %v", p, err)
			}
		} else {
			if err := fsw.Add(p); err != nil {
				log.Printf("warning: failed to watch %s: %v", p, err)
			}
		}
	}

	// Event processing loop with debouncing.
	var timer *time.Timer
	for {
		select {
		case event, ok := <-fsw.Events:
			if !ok {
				return nil
			}
			// Only trigger on write, create, remove, and rename events.
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
				continue
			}

			// If a new directory is created, watch it recursively.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					_ = w.addRecursive(event.Name)
				}
			}

			// Reset debounce timer.
			if timer != nil {
				timer.Stop()
			}
			timer = time.AfterFunc(w.debounce, func() {
				w.onChange()
			})

		case err, ok := <-fsw.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)

		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return fsw.Close()
		}
	}
}

// Stop signals the watcher to stop monitoring files.
func (w *Watcher) Stop() {
	w.once.Do(func() {
		close(w.done)
	})
}

// addRecursive adds a directory and all its subdirectories to the watcher.
func (w *Watcher) addRecursive(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := w.watcher.Add(path); err != nil {
				return err
			}
		}
		return nil
	})
}
