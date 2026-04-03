//go:build windows

package appicon

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var cached *walk.Icon

// Get returns the application icon loaded from the embedded Windows resource
// (embedded via rsrc.syso with resource ID 1). Returns nil on failure.
// The icon is cached after the first load.
func Get() *walk.Icon {
	if cached != nil {
		return cached
	}
	hIcon := win.LoadIcon(win.GetModuleHandle(nil), win.MAKEINTRESOURCE(1))
	if hIcon == 0 {
		return nil
	}
	icon, err := walk.NewIconFromHICON(hIcon)
	if err != nil {
		return nil
	}
	cached = icon
	return icon
}
