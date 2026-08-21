package config

import (
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"minigate/internal/logger"
	"minigate/internal/model"
	"minigate/internal/timeutil"
)

type ApplyFunc func(*model.GatewayConfig) error

type Reloader struct {
	src    Source
	apply  ApplyFunc
	mu     sync.RWMutex
	hash   string
	okAt   time.Time
	errMsg string
}

func NewReloader(src Source, apply ApplyFunc) *Reloader {
	return &Reloader{src: src, apply: apply}
}

func (r *Reloader) Status() model.HotReloadStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return model.HotReloadStatus{
		Source:      r.src.Name(),
		LastSuccess: timeutil.Format(r.okAt),
		LastError:   r.errMsg,
	}
}

func (r *Reloader) LoadAndApply() (*model.GatewayConfig, error) {
	cfg, err := r.src.Load()
	if err != nil {
		r.setErr(err.Error())
		return nil, err
	}
	if err := r.commit(cfg); err != nil {
		r.setErr(err.Error())
		return nil, err
	}
	return cfg, nil
}

func (r *Reloader) SaveAndApply(cfg *model.GatewayConfig) error {
	if err := r.src.Save(cfg); err != nil {
		r.setErr(err.Error())
		return err
	}
	return r.commit(cfg)
}

func (r *Reloader) commit(cfg *model.GatewayConfig) error {
	h := Hash(cfg)
	r.mu.Lock()
	defer r.mu.Unlock()
	if h == r.hash {
		return nil
	}
	if err := r.apply(cfg); err != nil {
		return err
	}
	r.hash = h
	r.okAt = timeutil.Now()
	r.errMsg = ""
	logger.L().Info("config applied", slog.String("source", r.src.Name()), slog.String("hash", h[:12]))
	return nil
}

func (r *Reloader) setErr(msg string) {
	r.mu.Lock()
	r.errMsg = msg
	r.mu.Unlock()
	logger.L().Error("config apply failed", slog.String("err", msg))
}

func (r *Reloader) WatchFile(path string, stop <-chan struct{}) error {
	fs, ok := r.src.(*FileSource)
	if !ok {
		return nil
	}
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	dir := dirOf(path)
	if err := w.Add(dir); err != nil {
		_ = w.Close()
		return err
	}
	go func() {
		defer w.Close()
		var timer *time.Timer
		for {
			select {
			case <-stop:
				return
			case ev, ok := <-w.Events:
				if !ok {
					return
				}
				if ev.Name != "" && !isSameFile(ev.Name, fs.Path) && !isSameFile(ev.Name, path) {
					continue
				}
				if timer != nil {
					timer.Stop()
				}
				timer = time.AfterFunc(300*time.Millisecond, func() {
					if _, err := r.LoadAndApply(); err != nil {
						logger.L().Warn("hot reload rejected", slog.String("err", err.Error()))
					}
				})
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				logger.L().Warn("fsnotify", slog.String("err", err.Error()))
			}
		}
	}()
	return nil
}

func dirOf(path string) string {
	i := stringsLastSlash(path)
	if i <= 0 {
		return "."
	}
	return path[:i]
}

func stringsLastSlash(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return i
		}
	}
	return -1
}

func isSameFile(a, b string) bool {
	ba := filepath.Base(a)
	bb := filepath.Base(b)
	return ba == bb || ba == bb+".tmp" || bb == ba+".tmp"
}
