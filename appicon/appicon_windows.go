//go:build windows

package appicon

import (
	"os"

	"github.com/lxn/walk"

	"github.com/athened/reborn-plugin-autoinstaller/logger"
	"github.com/athened/reborn-plugin-autoinstaller/resources"
)

var cached *walk.Icon

// Get returns the application icon, loading it from the embedded ICO bytes.
// The icon is written to a temp file once, loaded by walk, then the temp file
// is deleted (walk holds the HICON in memory after loading).
// Returns nil on failure — callers should always guard against nil.
func Get() *walk.Icon {
	if cached != nil {
		return cached
	}

	tmp, err := os.CreateTemp("", "rpa_icon_*.ico")
	if err != nil {
		logger.Error("appicon: create temp file: %v", err)
		return nil
	}
	tmpPath := tmp.Name()

	if _, err := tmp.Write(resources.Icon); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		logger.Error("appicon: write icon bytes: %v", err)
		return nil
	}
	tmp.Close()

	icon, err := walk.NewIconFromFile(tmpPath)
	os.Remove(tmpPath) // safe to delete after walk has loaded the HICON
	if err != nil {
		logger.Error("appicon: load icon from file: %v", err)
		return nil
	}

	logger.Debug("appicon: loaded successfully")
	cached = icon
	return icon
}
