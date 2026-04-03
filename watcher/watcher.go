//go:build windows

package watcher

import (
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/athened/reborn-plugin-autoinstaller/config"
	"github.com/athened/reborn-plugin-autoinstaller/installer"
	"github.com/athened/reborn-plugin-autoinstaller/logger"
)

// installCooldown is the minimum time that must elapse between two installs.
// This prevents a feedback loop: our own copyFile() write to the destination
// file emits a fsnotify Write event — without the cooldown that would
// immediately trigger another install, and so on forever.
const installCooldown = 3 * time.Second

type watcherState struct {
	mu            sync.Mutex
	fw            *fsnotify.Watcher
	running       bool
	lastInstallAt time.Time
}

var state *watcherState

// Start begins watching the destination .dat file for external changes.
// onResult is called (via the pump window's Synchronize) after each install attempt.
func Start(cfg *config.Config, onResult func(err error)) error {
	Stop()

	destPath := installer.DestPath(cfg.GameDir, cfg.PluginLang)
	logger.Info("watcher: starting watch on %s", destPath)

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("watcher: create fsnotify watcher: %v", err)
		return err
	}

	if err := fw.Add(destPath); err != nil {
		fw.Close()
		logger.Error("watcher: add path %s: %v", destPath, err)
		return err
	}

	ws := &watcherState{fw: fw, running: true}
	state = ws

	go ws.loop(cfg, onResult)
	logger.Debug("watcher: watch loop started")
	return nil
}

func (ws *watcherState) loop(cfg *config.Config, onResult func(err error)) {
	for {
		select {
		case event, ok := <-ws.fw.Events:
			if !ok {
				logger.Debug("watcher: events channel closed")
				return
			}

			logger.Debug("watcher: event op=%v file=%s", event.Op, event.Name)

			// Only react to Write or Create (game client overwrites / replaces the file)
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			// Cooldown guard — skip events caused by our own previous install.
			// When copyFile() writes to the destination, fsnotify fires a Write
			// event on that same file. Without this guard we'd loop forever.
			elapsed := time.Since(ws.lastInstallAt)
			if elapsed < installCooldown {
				logger.Debug("watcher: skipping event (cooldown, %v remaining)",
					installCooldown-elapsed)
				continue
			}

			logger.Info("watcher: destination file changed, reinstalling plugin")
			ws.lastInstallAt = time.Now()

			err := installer.Install(cfg)
			if err != nil {
				logger.Error("watcher: install failed: %v", err)
			} else {
				logger.Info("watcher: install succeeded")
			}
			if onResult != nil {
				onResult(err)
			}

		case err, ok := <-ws.fw.Errors:
			if !ok {
				logger.Debug("watcher: error channel closed")
				return
			}
			logger.Error("watcher: fsnotify error: %v", err)
		}
	}
}

// Stop closes the active watcher if any.
func Stop() {
	if state == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.running {
		logger.Debug("watcher: stopping")
		state.fw.Close()
		state.running = false
	}
	state = nil
}
