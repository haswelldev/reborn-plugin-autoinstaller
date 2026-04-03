//go:build windows

package ui

import (
	"fmt"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"

	"github.com/athened/reborn-plugin-autoinstaller/appicon"
	"github.com/athened/reborn-plugin-autoinstaller/config"
	"github.com/athened/reborn-plugin-autoinstaller/installer"
	"github.com/athened/reborn-plugin-autoinstaller/logger"
)

// SetupResult is returned by RunSetupWizard.
type SetupResult struct {
	Config    *config.Config
	Cancelled bool
}

// RunSetupWizard shows the 3-step setup wizard. Blocks until closed.
func RunSetupWizard(existing *config.Config) SetupResult {
	cfg := *existing
	result := SetupResult{Cancelled: true}

	var mw *walk.MainWindow
	var pages *walk.TabWidget

	var gameDirEdit *walk.LineEdit
	var gameDirError *walk.Label
	var pluginList *walk.ListBox
	var pluginError *walk.Label
	var modeClickRadio, modeTrayRadio *walk.RadioButton
	var autoStartupCB *walk.CheckBox
	var autoStartupGroup *walk.Composite
	var statusLabel *walk.Label
	var nextBtn, backBtn, finishBtn *walk.PushButton

	plugins := []installer.PluginInfo{}
	selectedPlugin := 0
	currentStep := 0

	goTo := func(step int) {
		currentStep = step
		pages.SetCurrentIndex(step)
		backBtn.SetVisible(step > 0)
		nextBtn.SetVisible(step < 2)
		finishBtn.SetVisible(step == 2)
		statusLabel.SetText("")
	}

	var doNext func()

	_ = MainWindow{
		AssignTo: &mw,
		Title:    "Reborn Plugin Autoinstaller — Setup",
		Size:     Size{Width: 520, Height: 390},
		MinSize:  Size{Width: 520, Height: 390},
		Layout:   VBox{Margins: Margins{Left: 12, Top: 8, Right: 12, Bottom: 8}},
		Children: []Widget{
			TabWidget{
				AssignTo: &pages,
				Pages: []TabPage{
					// ── Step 1: Game Folder ──────────────────────────────────
					{
						Title:  "1. Game Folder",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Select your Lineage Reborn game folder."},
							Label{
								Text:      "Hint: for Signature, it is usually  Reborn/games/signature",
								TextColor: walk.RGB(100, 100, 100),
							},
							VSpacer{Size: 6},
							Composite{
								Layout: HBox{MarginsZero: true},
								Children: []Widget{
									LineEdit{
										AssignTo: &gameDirEdit,
										Text:     cfg.GameDir,
										OnTextChanged: func() {
											cfg.GameDir = gameDirEdit.Text()
										},
									},
									PushButton{
										Text:    "Browse…",
										MaxSize: Size{Width: 80},
										OnClicked: func() {
											p := browseFolderDialog(mw, cfg.GameDir)
											if p != "" {
												cfg.GameDir = p
												gameDirEdit.SetText(p)
											}
										},
									},
								},
							},
							Label{
								AssignTo:  &gameDirError,
								TextColor: walk.RGB(200, 0, 0),
								Visible:   false,
							},
						},
					},
					// ── Step 2: Plugin ───────────────────────────────────────
					{
						Title:  "2. Plugin",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "Select the plugin to use. Language is detected automatically from the file."},
							VSpacer{Size: 4},
							ListBox{
								AssignTo: &pluginList,
								OnCurrentIndexChanged: func() {
									idx := pluginList.CurrentIndex()
									if idx >= 0 && idx < len(plugins) {
										selectedPlugin = idx
										cfg.PluginName = plugins[idx].Name
										cfg.PluginLang = plugins[idx].LangCode
									}
								},
							},
							Label{
								AssignTo:  &pluginError,
								TextColor: walk.RGB(200, 60, 0),
								Visible:   false,
							},
						},
					},
					// ── Step 3: Run Mode ─────────────────────────────────────
					{
						Title:  "3. Run Mode",
						Layout: VBox{},
						Children: []Widget{
							Label{Text: "How should the app work?"},
							VSpacer{Size: 6},
							RadioButton{
								AssignTo: &modeClickRadio,
								Text:     "Click Mode — install on each app launch, then show settings",
								OnClicked: func() {
									cfg.RunMode = "click"
									autoStartupGroup.SetVisible(false)
								},
							},
							RadioButton{
								AssignTo: &modeTrayRadio,
								Text:     "Tray Mode — run in background, auto-reinstall on game updates",
								OnClicked: func() {
									cfg.RunMode = "tray"
									autoStartupGroup.SetVisible(true)
								},
							},
							Composite{
								AssignTo: &autoStartupGroup,
								Layout:   HBox{MarginsZero: true},
								Visible:  false,
								Children: []Widget{
									HSpacer{Size: 20},
									CheckBox{
										AssignTo: &autoStartupCB,
										Text:     "Start with Windows",
										OnCheckedChanged: func() {
											cfg.AutoStartup = autoStartupCB.Checked()
										},
									},
								},
							},
							VSpacer{},
							Label{Text: "You can change this later in Settings.", TextColor: walk.RGB(120, 120, 120)},
						},
					},
				},
			},

			Label{AssignTo: &statusLabel, TextColor: walk.RGB(180, 0, 0)},

			Composite{
				Layout: HBox{MarginsZero: true},
				Children: []Widget{
					HSpacer{},
					PushButton{
						AssignTo:  &backBtn,
						Text:      "← Back",
						Visible:   false,
						OnClicked: func() { goTo(currentStep - 1) },
					},
					PushButton{
						AssignTo:  &nextBtn,
						Text:      "Next →",
						OnClicked: func() { doNext() },
					},
					PushButton{
						AssignTo: &finishBtn,
						Text:     "Finish",
						Visible:  false,
						OnClicked: func() {
							cfg.Configured = true
							result = SetupResult{Config: &cfg}
							logger.Info("setup wizard: finished, mode=%s plugin=%s lang=%s",
								cfg.RunMode, cfg.PluginName, cfg.PluginLang)
							mw.Close()
						},
					},
					PushButton{
						Text:      "Cancel",
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

	// Default run mode
	if cfg.RunMode == "tray" {
		modeTrayRadio.SetChecked(true)
		autoStartupGroup.SetVisible(true)
		autoStartupCB.SetChecked(cfg.AutoStartup)
	} else {
		modeClickRadio.SetChecked(true)
		cfg.RunMode = "click"
	}

	doNext = func() {
		statusLabel.SetText("")
		switch currentStep {
		case 0: // validate game dir → scan plugins → go to step 1
			dir := gameDirEdit.Text()
			if dir == "" {
				gameDirError.SetText("Please select your game folder.")
				gameDirError.SetVisible(true)
				return
			}
			logger.Debug("setup: validating game dir: %s", dir)
			if !installer.ValidateGameDir(dir) {
				gameDirError.SetText("Not a valid game folder (system/lang/e/SystemMsg-e.dat not found).")
				gameDirError.SetVisible(true)
				return
			}
			gameDirError.SetVisible(false)
			cfg.GameDir = dir

			found, badStructure, err := installer.ScanPlugins(dir)
			if err != nil {
				statusLabel.SetText(fmt.Sprintf("Error scanning plugins: %v", err))
				logger.Error("setup: scan plugins error: %v", err)
				return
			}
			logger.Info("setup: found %d plugins, badStructure=%v", len(found), badStructure)
			plugins = found

			model := make([]string, len(plugins))
			for i, p := range plugins {
				model[i] = p.DisplayName
			}
			pluginList.SetModel(model)

			switch {
			case len(plugins) == 0 && badStructure:
				pluginError.SetText(
					"Reborn plugin file structure might have changed.\n" +
						"Check for an app update — it may already be fixed.")
				pluginError.SetVisible(true)
			case badStructure:
				pluginError.SetText("Some plugin folders had unexpected structure and were skipped.")
				pluginError.SetVisible(true)
			default:
				pluginError.SetVisible(false)
			}

			// Restore previous selection if it still exists
			for i, p := range plugins {
				if p.Name == existing.PluginName && p.LangCode == existing.PluginLang {
					selectedPlugin = i
				}
			}
			if len(plugins) > 0 {
				pluginList.SetCurrentIndex(selectedPlugin)
				cfg.PluginName = plugins[selectedPlugin].Name
				cfg.PluginLang = plugins[selectedPlugin].LangCode
			}
			goTo(1)

		case 1: // validate plugin selection → go to step 2
			if len(plugins) == 0 {
				statusLabel.SetText("No plugins found. Check your game folder or look for an app update.")
				return
			}
			cfg.PluginName = plugins[selectedPlugin].Name
			cfg.PluginLang = plugins[selectedPlugin].LangCode
			goTo(2)
		}
	}

	pages.SetCurrentIndex(0)
	mw.Run()
	return result
}
