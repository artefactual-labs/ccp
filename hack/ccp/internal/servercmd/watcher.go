package servercmd

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/go-logr/logr"
	"github.com/gohugoio/hugo/watcher"

	"github.com/artefactual/archivematica/hack/ccp/internal/workflow"
)

type observer interface {
	Notify(path string) error
}

func watch(logger logr.Logger, o observer, wf *workflow.Document, path string) (*watcher.Batcher, error) {
	w, err := watcher.New(500*time.Millisecond, 700*time.Millisecond, false)
	if err != nil {
		return nil, err
	}

	var errs error
	for _, wd := range wf.WatchedDirectories {
		wdPath := filepath.Join(path, wd.Path)
		info, err := os.Stat(wdPath)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if !info.IsDir() {
			errs = errors.Join(errs, errors.New("not a directory"))
			continue
		}
		logger.V(2).Info("Watching directory.", "path", wdPath)
		if err := w.Add(wdPath); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
	}
	if errs != nil {
		return nil, errs
	}

	go func() {
		for {
			select {
			case evs := <-w.Events:
				notify(logger, o, evs)
			case err := <-w.Errors():
				if err != nil {
					logger.V(1).Info("Error while watching.", "err", err)
				}
				return
			}
		}
	}()

	return w, nil
}

func notify(logger logr.Logger, o observer, evs []fsnotify.Event) {
	for _, ev := range evs {
		if ev.Op&fsnotify.Create != fsnotify.Create {
			continue
		}
		if err := o.Notify(ev.Name); err != nil {
			logger.Error(err, "Failed to notify controller with a new event.", "err", err, "path", ev.Name)
		}
	}
}
