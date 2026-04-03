package resources

import _ "embed"

// Icon holds the raw bytes of the application icon (ICO format).
// Embedded at compile time so it is always available without touching the filesystem.
//
//go:embed icon.ico
var Icon []byte
