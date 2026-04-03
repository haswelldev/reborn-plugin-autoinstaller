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
	// Prevents two instances running in parallel (duplicate tray icons etc.)
	mutexName, _ := windows.UTF16PtrFromString("RebornPluginAutoinstallerMutex")
	mutex, mutexErr := windows.CreateMutex(nil, false, mutexName)
	if mutexErr == windows.ERROR_ALREADY_EXISTS {
		// Another instance is already running
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

	// ── Click Mode ────────────────────────────────────────────────────────
	if cfg.RunMode == "click" {
		logger.Info("mode: click")
		runClickMode(cfg)
		return
	}

	// ── Tray Mode ─────────────────────────────────────────────────────────
	logger.Info("mode: tray")
	runTrayMode(cfg)
}

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

func runTrayMode(cfg *config.Config) {
	pumpWindow, err := walk.NewMainWindow()
	if err != nil {
		log.Fatalf("failed to create pump window: %v", err)
	}
	pumpWindow.SetVisible(false)
	logger.Debug("tray: pump window created")

	settingsOpen := false

	openSettings := func() {
		if settingsOpen {
			return
		}
		settingsOpen = true
		ui.RunSettingsWindow(cfg, nil, func(newCfg *config.Config) error {
			prevLang := cfg.PluginLang
			*cfg = *newCfg
			if err := cfg.Save(); err != nil {
				return err
			}
			// Restart watcher if the watched path changed
			if cfg.PluginLang != prevLang || cfg.GameDir != newCfg.GameDir {
				watcher.Stop()
				if cfg.RunMode == "tray" {
					if err := startWatcher(pumpWindow, cfg); err != nil {
						logger.Error("tray: watcher restart: %v", err)
					}
				}
			}
			return nil
		})
		settingsOpen = false
	}

	quit := func() {
		logger.Info("tray: quitting")
		watcher.Stop()
		tray.Dispose()
		pumpWindow.Close()
	}

	if err := tray.Init(pumpWindow, openSettings, quit); err != nil {
		log.Fatalf("tray init: %v", err)
	}
	logger.Info("tray: icon initialized")

	if err := startWatcher(pumpWindow, cfg); err != nil {
		logger.Error("tray: initial watcher start: %v", err)
		// Non-fatal: show notification so user knows
		tray.ShowBalloon("Watch Error",
			"Could not watch plugin file: "+err.Error()+"\nCheck Settings.")
	}

	pumpWindow.Run()
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
