// Package watcher watches the filesystem for changes and triggers callbacks.
package watcher

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeType describes what kind of file event occurred.
type ChangeType int

const (
	ChangeModified ChangeType = iota
	ChangeCreated
	ChangeDeleted
)

// Event represents a file system change event.
type Event struct {
	Path string
	Type ChangeType
}

// Watcher wraps fsnotify and provides a debounced event stream.
type Watcher struct {
	fw       *fsnotify.Watcher
	onChange func(Event)
	debounce time.Duration
}

// New creates a new file watcher with a debounce duration.
func New(debounce time.Duration, onChange func(Event)) (*Watcher, error) {
	fw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{fw: fw, onChange: onChange, debounce: debounce}, nil
}

// Watch adds a directory (recursively) to the watch list.
func (w *Watcher) Watch(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return w.fw.Add(path)
		}
		return nil
	})
}

// Start begins listening for file system events in a background goroutine.
func (w *Watcher) Start() {
	go func() {
		var (
			timer   *time.Timer
			lastEvt Event
		)
		for {
			select {
			case ev, ok := <-w.fw.Events:
				if !ok {
					return
				}
				lastEvt = toEvent(ev)
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(w.debounce, func() {
					w.onChange(lastEvt)
				})
			case err, ok := <-w.fw.Errors:
				if !ok {
					return
				}
				log.Printf("[GoKS/watcher] error: %v", err)
			}
		}
	}()
}

// Close stops the watcher.
func (w *Watcher) Close() {
	w.fw.Close()
}

func toEvent(ev fsnotify.Event) Event {
	t := ChangeModified
	if ev.Has(fsnotify.Create) {
		t = ChangeCreated
	} else if ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
		t = ChangeDeleted
	}
	return Event{Path: ev.Name, Type: t}
}
