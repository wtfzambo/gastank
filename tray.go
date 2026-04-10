package main

import _ "embed"

//go:embed tray_icon.png
var trayIcon []byte

// trayIconBytes returns the embedded 64×64 tray icon PNG.
// This is a black silhouette on transparent background, suitable
// for macOS template rendering (auto-adapts to dark/light mode).
func trayIconBytes() []byte {
	return trayIcon
}
