//go:build windows

package tray

import (
	"fmt"
	"time"

	"github.com/lxn/walk"

	"github.com/athened/reborn-plugin-autoinstaller/appicon"
)

var notifyIcon *walk.NotifyIcon

// Init creates the system tray icon attached to the given (hidden) parent window.
func Init(parent *walk.MainWindow, openSettings func(), quit func()) error {
	ni, err := walk.NewNotifyIcon(parent)
	if err != nil {
		return fmt.Errorf("create notify icon: %w", err)
	}
	notifyIcon = ni

	if icon := appicon.Get(); icon != nil {
		ni.SetIcon(icon)
	}

	ni.SetToolTip("Reborn Plugin Autoinstaller")
	ni.SetVisible(true)

	// Left-click opens settings
	ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			openSettings()
		}
	})

	// Context menu
	openAction := walk.NewAction()
	openAction.SetText("Open Settings")
	openAction.Triggered().Attach(openSettings)

	quitAction := walk.NewAction()
	quitAction.SetText("Quit")
	quitAction.Triggered().Attach(quit)

	ni.ContextMenu().Actions().Add(openAction)
	ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())
	ni.ContextMenu().Actions().Add(quitAction)

	return nil
}

// SetTooltip updates the tray icon tooltip.
func SetTooltip(text string) {
	if notifyIcon != nil {
		notifyIcon.SetToolTip(text)
	}
}

// SetLastInstalled updates the tooltip with the last install time.
func SetLastInstalled(t time.Time) {
	SetTooltip(fmt.Sprintf("Reborn Plugin Autoinstaller\nLast installed: %s", t.Format("15:04:05")))
}

// ShowBalloon displays a balloon notification from the tray icon.
func ShowBalloon(title, msg string) {
	if notifyIcon != nil {
		notifyIcon.ShowInfo(title, msg)
	}
}

// Dispose cleans up the notify icon.
func Dispose() {
	if notifyIcon != nil {
		notifyIcon.Dispose()
		notifyIcon = nil
	}
}
