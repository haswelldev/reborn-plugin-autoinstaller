//go:build windows

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lxn/walk"
	"golang.org/x/sys/windows"

	"github.com/athened/reborn-plugin-autoinstaller/autostart"
	"github.com/athened/reborn-plugin-autoinstaller/config"
	"github.com/athened/reborn-plugin-autoinstaller/installer"
	"github.com/athened/reborn-plugin-autoinstaller/logger"
	"github.com/athened/reborn-plugin-autoinstaller/tray"
	"github.com/athened/reborn-plugin-autoinstaller/ui"
	"github.com/athened/reborn-plugin-autoinstaller/watcher"
)

func main() {
	logger.Init()
	logger.Info("startup: pid=%d", os.Getpid())

	// ── Single-instance lock ──────────────────────────────────────────────
	mutexName, _ := windows.UTF16PtrFromString("RebornPluginAutoinstallerMutex")
	mutex, mutexErr := windows.CreateMutex(nil, false, mutexName)
	if mutexErr == windows.ERROR_ALREADY_EXISTS {
		logger.Warn("startup: another instance detected, exiting")
		if mutex != 0 {
			windows.CloseHandle(mutex)
		}
		walk.MsgBox(nil, "Already Running",
			"Reborn Plugin Autoinstaller is already running.\nCheck your system tray.",
			walk.MsgBoxIconInformation)
		os.Exit(0)
	}
	if mutexErr != nil {
		log.Printf("CreateMutex error (non-fatal): %v", mutexErr)
	}
	if mutex != 0 {
		defer windows.CloseHandle(mutex)
	}

	// ── Load config ───────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger.Info("config: dir=%s plugin=%s lang=%s mode=%s configured=%v",
		cfg.GameDir, cfg.PluginName, cfg.PluginLang, cfg.RunMode, cfg.Configured)

	// ── First run: setup wizard ───────────────────────────────────────────
	if !cfg.Configured {
		logger.Info("startup: first run, showing setup wizard")
		res := ui.RunSetupWizard(cfg)
		if res.Cancelled || res.Config == nil {
			logger.Info("startup: setup cancelled")
			os.Exit(0)
		}
		cfg = res.Config
		if err := cfg.Save(); err != nil {
			walk.MsgBox(nil, "Error", "Failed to save settings:\n"+err.Error(), walk.MsgBoxIconError)
			os.Exit(1)
		}
		logger.Info("startup: setup complete, mode=%s plugin=%s lang=%s",
			cfg.RunMode, cfg.PluginName, cfg.PluginLang)

		if cfg.RunMode == "tray" && cfg.AutoStartup {
			if exePath, err := os.Executable(); err == nil {
				if err := autostart.Enable(exePath); err != nil {
					logger.Error("autostart enable: %v", err)
				} else {
					logger.Info("autostart enabled: %s", exePath)
				}
			}
		}
	}

	// ── Dispatch by mode ──────────────────────────────────────────────────
	if cfg.RunMode == "click" {
		logger.Info("mode: click")
		runClickMode(cfg)
	} else {
		logger.Info("mode: tray")
		runTrayMode(cfg)
	}

	// Ensure the process always terminates cleanly.
	// (CGO + walk message pump can occasionally leave stale goroutines.)
	os.Exit(0)
}

// ── Click Mode ────────────────────────────────────────────────────────────────

func runClickMode(cfg *config.Config) {
	logger.Info("click: installing %s (lang=%s)", cfg.PluginName, cfg.PluginLang)
	installErr := installer.Install(cfg)

	var initial *ui.InitialStatus
	if installErr != nil {
		logger.Error("click: install failed: %v", installErr)
		initial = &ui.InitialStatus{
			Message: fmt.Sprintf("Install failed: %v", installErr),
			IsError: true,
		}
	} else {
		msg := installer.DisplayName(cfg.PluginName, cfg.PluginLang) + " was installed!"
		logger.Info("click: %s", msg)
		initial = &ui.InitialStatus{Message: msg, IsError: false}
	}

	ui.RunSettingsWindow(cfg, initial, func(newCfg *config.Config) error {
		*cfg = *newCfg
		return cfg.Save()
	})
}

// ── Tray Mode ─────────────────────────────────────────────────────────────────

func runTrayMode(cfg *config.Config) {
	pumpWindow, err := walk.NewMainWindow()
	if err != nil {
		log.Fatalf("failed to create pump window: %v", err)
	}
	pumpWindow.SetVisible(false)
	logger.Debug("tray: pump window created")

	settingsOpen := false
	switchedToClick := false // set when user changes mode → click inside settings

	openSettings := func() {
		if settingsOpen {
			return
		}
		settingsOpen = true

		// Capture mode at the moment settings window opens.
		modeBeforeOpen := cfg.RunMode

		ui.RunSettingsWindow(cfg, nil, func(newCfg *config.Config) error {
			*cfg = *newCfg
			if err := cfg.Save(); err != nil {
				return err
			}
			// Always re-evaluate the watcher on save (mode, lang or dir may have changed).
			watcher.Stop()
			if cfg.RunMode == "tray" {
				if err := startWatcher(pumpWindow, cfg); err != nil {
					logger.Error("tray: watcher restart: %v", err)
				}
			}
			return nil
		})

		settingsOpen = false

		// After the settings window is closed, check whether the user switched
		// from Tray → Click. If so, tear everything down and fall through to
		// runClickMode() after the pump exits.
		if modeBeforeOpen == "tray" && cfg.RunMode == "click" {
			logger.Info("tray: mode switched to click, tearing down tray")
			switchedToClick = true
			watcher.Stop()
			tray.Dispose()
			// Schedule pump closure on the UI goroutine.
			pumpWindow.Synchronize(func() { pumpWindow.Close() })
		}
	}

	// Quit: stop everything and exit the process immediately.
	quit := func() {
		logger.Info("tray: quit requested")
		watcher.Stop()
		tray.Dispose()
		os.Exit(0) // os.Exit is the only reliable way to terminate when CGO is involved
	}

	if err := tray.Init(pumpWindow, openSettings, quit); err != nil {
		log.Fatalf("tray init: %v", err)
	}
	logger.Info("tray: icon initialized")

	if err := startWatcher(pumpWindow, cfg); err != nil {
		logger.Error("tray: initial watcher start: %v", err)
		tray.ShowBalloon("Watch Error",
			"Could not watch plugin file: "+err.Error()+"\nCheck Settings.")
	}

	pumpWindow.Run() // blocks until pump window closes

	// If the user switched to Click mode inside Settings, run click mode now
	// (tray icon and watcher are already stopped above).
	if switchedToClick {
		logger.Info("tray→click: running click mode")
		runClickMode(cfg)
	}
}

func startWatcher(pump *walk.MainWindow, cfg *config.Config) error {
	return watcher.Start(cfg, func(err error) {
		pump.Synchronize(func() {
			if err != nil {
				tray.ShowBalloon("Install Failed", err.Error())
			} else {
				msg := installer.DisplayName(cfg.PluginName, cfg.PluginLang) + " was installed!"
				tray.ShowBalloon("Plugin Installed", msg)
				tray.SetLastInstalled(time.Now())
			}
		})
	})
}
