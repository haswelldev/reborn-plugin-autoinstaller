package autostart

import (
	"golang.org/x/sys/windows/registry"
)

const (
	keyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName = "RebornPluginAutoinstaller"
)

// Enable registers the app to run at Windows startup.
func Enable(exePath string) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.SetStringValue(appName, exePath)
}

// Disable removes the app from Windows startup.
func Disable() error {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	return k.DeleteValue(appName)
}

// IsEnabled returns true if the startup entry exists.
func IsEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	return err == nil
}
