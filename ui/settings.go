//go:build windows

package ui

import (
	"fmt"
	"time"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"github.com/athened/reborn-plugin-autoinstaller/appicon"
	"github.com/athened/reborn-plugin-autoinstaller/autostart"
	"github.com/athened/reborn-plugin-autoinstaller/config"
	"github.com/athened/reborn-plugin-autoinstaller/installer"
	"github.com/athened/reborn-plugin-autoinstaller/logger"
)

// OnSettingsChange is called when the user saves new settings.
type OnSettingsChange func(newCfg *config.Config) error

// InitialStatus optionally shows a pre-set status message when the window opens.
type InitialStatus struct {
	Message string
	IsError bool
}

// RunSettingsWindow shows the settings window. Blocks until closed.
func RunSettingsWindow(cfg *config.Config, initial *InitialStatus, onChange OnSettingsChange) {
	cur := *cfg

	var mw *walk.MainWindow
	var gameDirEdit *walk.LineEdit
	var pluginList *walk.ListBox
	var modeClickRadio, modeTrayRadio *walk.RadioButton
	var autoStartupCB *walk.CheckBox
	var autoStartupGroup *walk.Composite
	var lastInstalledLabel *walk.Label
	var statusLabel *walk.Label
	var reInstallBtn *walk.PushButton

	plugins := []installer.PluginInfo{}
	selectedPlugin := 0

	refreshPlugins := func() {
		found, _, _ := installer.ScanPlugins(cur.GameDir)
		plugins = found
		model := make([]string, len(plugins))
		for i, p := range plugins {
			model[i] = p.DisplayName
		}
		pluginList.SetModel(model)
		for i, p := range plugins {
			if p.Name == cur.PluginName && p.LangCode == cur.PluginLang {
				selectedPlugin = i
				pluginList.SetCurrentIndex(i)
				return
			}
		}
		if len(plugins) > 0 {
			pluginList.SetCurrentIndex(0)
			selectedPlugin = 0
			cur.PluginName = plugins[0].Name
			cur.PluginLang = plugins[0].LangCode
		}
	}

	_ = MainWindow{
		AssignTo: &mw,
		Title:    "Reborn Plugin Autoinstaller — Settings",
		Size:     Size{Width: 480, Height: 480},
		MinSize:  Size{Width: 480, Height: 430},
		Layout:   VBox{Margins: Margins{Left: 16, Top: 12, Right: 16, Bottom: 12}},
		Children: []Widget{
			Label{Text: "Game Folder:"},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					LineEdit{
						AssignTo:      &gameDirEdit,
						Text:          cfg.GameDir,
						OnTextChanged: func() { cur.GameDir = gameDirEdit.Text() },
					},
					PushButton{
						Text:    "Browse…",
						MaxSize: Size{Width: 80},
						OnClicked: func() {
							p := browseFolderDialog(mw, cur.GameDir)
							if p != "" {
								cur.GameDir = p
								gameDirEdit.SetText(p)
								refreshPlugins()
							}
						},
					},
				},
			},

			VSpacer{Size: 8},
			Label{Text: "Plugin:"},
			ListBox{
				AssignTo: &pluginList,
				MaxSize:  Size{Height: 90},
				OnCurrentIndexChanged: func() {
					idx := pluginList.CurrentIndex()
					if idx >= 0 && idx < len(plugins) {
						selectedPlugin = idx
						cur.PluginName = plugins[idx].Name
						cur.PluginLang = plugins[idx].LangCode
					}
				},
			},

			VSpacer{Size: 8},
			Label{Text: "Run Mode:"},
			RadioButton{
				AssignTo: &modeClickRadio,
				Text:     "Click Mode (install on app launch)",
				OnClicked: func() {
					cur.RunMode = "click"
					autoStartupGroup.SetVisible(false)
				},
			},
			RadioButton{
				AssignTo: &modeTrayRadio,
				Text:     "Tray Mode (run in background, auto-install on game updates)",
				OnClicked: func() {
					cur.RunMode = "tray"
					autoStartupGroup.SetVisible(true)
				},
			},
			Composite{
				AssignTo: &autoStartupGroup,
				Layout:   HBox{MarginsZero: true},
				Visible:  cfg.RunMode == "tray",
				Children: []Widget{
					HSpacer{Size: 20},
					CheckBox{
						AssignTo:         &autoStartupCB,
						Text:             "Start with Windows",
						Checked:          cfg.AutoStartup,
						OnCheckedChanged: func() { cur.AutoStartup = autoStartupCB.Checked() },
					},
				},
			},

			VSpacer{Size: 8},
			Label{
				AssignTo:  &lastInstalledLabel,
				Text:      "Plugin not yet installed this session.",
				TextColor: walk.RGB(100, 100, 100),
			},
			Label{
				AssignTo:  &statusLabel,
				TextColor: walk.RGB(0, 130, 0),
			},

			VSpacer{},
			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					PushButton{
						AssignTo: &reInstallBtn,
						Text:     "Re-install Now",
						OnClicked: func() {
							reInstallBtn.SetEnabled(false)
							statusLabel.SetTextColor(walk.RGB(100, 100, 100))
							statusLabel.SetText("Installing…")
							go func() {
								err := installer.Install(&cur)
								mw.Synchronize(func() {
									reInstallBtn.SetEnabled(true)
									if err != nil {
										logger.Error("settings: re-install failed: %v", err)
										statusLabel.SetTextColor(walk.RGB(200, 0, 0))
										statusLabel.SetText(fmt.Sprintf("Error: %v", err))
									} else {
										logger.Info("settings: re-install succeeded: %s lang=%s",
											cur.PluginName, cur.PluginLang)
										now := time.Now()
										statusLabel.SetTextColor(walk.RGB(0, 140, 0))
										statusLabel.SetText(installer.DisplayName(cur.PluginName, cur.PluginLang) + " was installed!")
										lastInstalledLabel.SetText("Last installed: " + now.Format("15:04:05"))
									}
								})
							}()
						},
					},
					HSpacer{},
					PushButton{
						Text: "Save",
						OnClicked: func() {
							if err := onChange(&cur); err != nil {
								walk.MsgBox(mw, "Error", fmt.Sprintf("Failed to save: %v", err), walk.MsgBoxIconError)
								return
							}
							// Handle auto-startup setting
							if cur.RunMode == "tray" {
								if cur.AutoStartup {
									exePath, _ := exePathForAutostart()
									if err := autostart.Enable(exePath); err != nil {
										logger.Error("settings: autostart enable: %v", err)
									}
								} else {
									autostart.Disable()
								}
							} else {
								autostart.Disable()
							}
							*cfg = cur
							statusLabel.SetTextColor(walk.RGB(0, 140, 0))
							statusLabel.SetText("Settings saved.")
						},
					},
					PushButton{
						Text:      "Close",
						OnClicked: func() { mw.Close() },
					},
				},
			},
		},
	}.Create()

	// Set app icon on the window (taskbar)
	if icon := appicon.Get(); icon != nil {
		mw.SetIcon(icon)
	}

	if cfg.RunMode == "tray" {
		modeTrayRadio.SetChecked(true)
	} else {
		modeClickRadio.SetChecked(true)
	}

	refreshPlugins()

	// Show initial status if provided
	if initial != nil && initial.Message != "" {
		if initial.IsError {
			statusLabel.SetTextColor(walk.RGB(200, 0, 0))
		} else {
			statusLabel.SetTextColor(walk.RGB(0, 140, 0))
		}
		statusLabel.SetText(initial.Message)
		if !initial.IsError {
			lastInstalledLabel.SetText("Last installed: " + time.Now().Format("15:04:05"))
		}
	}

	_ = selectedPlugin
	mw.Run()
}
