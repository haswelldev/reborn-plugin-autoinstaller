//go:build windows

package ui

import (
	"os"

	"github.com/lxn/walk"
)

// browseFolderDialog opens a native folder picker dialog.
// Returns selected path or empty string if cancelled.
func browseFolderDialog(owner walk.Form, initial string) string {
	dlg := new(walk.FileDialog)
	dlg.Title = "Select Game Folder"
	if initial != "" {
		if _, err := os.Stat(initial); err == nil {
			dlg.InitialDirPath = initial
		}
	}
	ok, err := dlg.ShowBrowseFolder(owner)
	if err != nil || !ok {
		return ""
	}
	return dlg.FilePath
}

// exePathForAutostart returns the path of the running executable.
func exePathForAutostart() (string, error) {
	return os.Executable()
}
