package main

import (
	"embed"
	"log"
	"runtime"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := NewApp()

	wailsApp := application.New(application.Options{
		Name:        "Gastank",
		Description: "AI token usage monitor",
		Services: []application.Service{
			application.NewService(app),
		},
		Assets: application.AssetOptions{
			Handler: application.BundledAssetFileServer(assets),
		},
		Mac: application.MacOptions{
			// Accessory policy keeps the app out of the Dock and
			// removes it from Cmd+Tab, matching tray-app conventions.
			ActivationPolicy: application.ActivationPolicyAccessory,
		},
	})

	// Tray window — starts hidden, frameless, always on top.
	window := wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:           "Gastank",
		Width:           360,
		Height:          420,
		Hidden:          true,
		Frameless:       true,
		AlwaysOnTop:     true,
		DisableResize:   true,
		HideOnFocusLost: true,
		HideOnEscape:    true,
		Windows: application.WindowsWindow{
			HiddenOnTaskbar: true,
		},
	})

	// Closing the window hides instead of quitting.
	window.RegisterHook(events.Common.WindowClosing, func(e *application.WindowEvent) {
		window.Hide()
		e.Cancel()
	})

	// System tray — left-click toggles, right-click shows menu.
	tray := wailsApp.SystemTray.New()

	if runtime.GOOS == "darwin" {
		tray.SetTemplateIcon(trayIconBytes())
	} else {
		tray.SetIcon(trayIconBytes())
	}

	tray.SetTooltip("Gastank — AI usage monitor")
	tray.AttachWindow(window).WindowOffset(5)

	menu := wailsApp.Menu.New()
	menu.Add("Show").OnClick(func(_ *application.Context) {
		tray.ShowWindow()
	})
	menu.Add("Hide").OnClick(func(_ *application.Context) {
		tray.HideWindow()
	})
	menu.AddSeparator()
	menu.Add("Quit Gastank").OnClick(func(_ *application.Context) {
		wailsApp.Quit()
	})
	tray.SetMenu(menu)

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
